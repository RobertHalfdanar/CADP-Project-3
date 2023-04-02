package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	BroadcastTime    = 20 * time.Millisecond
	ElectionTimeout  = 20 * BroadcastTime
	RandomTimeoutMax = 300
	RandomTimeoutMin = 150
)

type RaftState int

const (
	Follower RaftState = iota
	Candidate
	Leader
	Failed
)

var command = make(chan string)
var hasFinishedExecutingCommand = make(chan bool)

func (e RaftState) String() string {
	switch e {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Failed:
		return "Failed"
	}
	panic("Unknown RaftState")
}

type Entry = Raft.LogEntry

type State struct {
	CurrentTerm uint64
	VotedFor    *net.UDPAddr
	Log         []*Entry

	CommitIndex uint64
	LastApplied uint64

	NextIndex  []uint64
	MatchIndex []uint64

	state RaftState // The state we are in

	MyName    string         // Given name
	MyAddress *net.UDPAddr   // The Address we listen on
	Listener  *net.UDPConn   // Connection we listen on
	Servers   []*net.UDPAddr // List of all servers

	votes             int                          // Did the server get votes?
	leftToVote        []*net.UDPAddr               // How has voted
	lastAppendRequest []*Raft.AppendEntriesRequest // Did we get a response?

	leader *net.UDPAddr

	lock sync.RWMutex
}

// resetToFollower resets the state of the server to follower.
// It also resets the term to the given term.
func (state *State) resetToFollower(newTerm uint64) {
	state.state = Follower
	state.CurrentTerm = newTerm
	state.VotedFor = nil
	state.MatchIndex = make([]uint64, len(state.Servers))
	state.NextIndex = make([]uint64, len(state.Servers))
}

// setServers reads a configuration file and populates the server array.
// It also checks if itself is in the configuration file.
// The parameter is the name of the configuration file.
func (state *State) setServers(fileName string) {
	file, err := os.OpenFile("./"+fileName, os.O_RDONLY, 0644)

	// If we receive an error closing the files, it will probably be because the file is already closed,
	// and can then ignore the error
	defer file.Close()

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to locate config file!")
		panic("Failed to locate config file")
	}

	isInFile := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		address := scanner.Text()

		state.Servers = append(state.Servers, Utils.CreateUDPAddr(address))

		if address == state.MyName {
			isInFile = true
		}
	}

	if isInFile == false {
		Logger.Log(Logger.ERROR, "Self not in configuration file!")
		panic("Self not in configuration file")
	}
}

// Init initialises the state of the server.
func (state *State) Init() {
	Logger.Log(Logger.INFO, "Initialising state of server...")
	defer Logger.Log(Logger.INFO, "Server initialized")

	if len(os.Args) < 3 {
		Logger.Log(Logger.ERROR, "Too few arguments")
		panic("Too few arguments")
	} else if len(os.Args) > 3 {
		Logger.Log(Logger.ERROR, "Too few arguments")
		panic("Too many arguments")
	}

	state.MyName = os.Args[1]
	state.MyAddress = Utils.CreateUDPAddr(state.MyName)

	fileName := os.Args[2]
	state.setServers(fileName)

	var err error
	state.Listener, err = Utils.CreateUDPListener(state.MyName)
	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to create a udp listener!")
		panic("Failed to create a udp listener!")
	}
}

// commitEntry will apply all entries up to the commit index.
// When server applies an entry, it will log the comment
func (state *State) commitEntry() {
	Logger.Log(Logger.INFO, "Committing entry...")

	// Our implementation of Raft LastApplied is a pointer to the last log in the file.
	// It is always trying to catch up to the commit index.
	// This is because if the server has been suspended for a while, it might have missed some entries.
	// And needs to receive them before committing and then applying them.

	// If commitIndex > lastApplied: increment lastApplied, apply log[lastApplied] to state machine (§5.3)
	for ; state.LastApplied < state.CommitIndex && state.LastApplied < uint64(len(state.Log)); state.LastApplied++ {
		log.Println(fmt.Sprintf("%d,%d,%s", state.Log[state.LastApplied].Term, state.Log[state.LastApplied].Index, state.Log[state.LastApplied].CommandName))
	}
}

// sendHeartbeat sends a heartbeat to all other servers.
// If a server is missing a log entry, it will send one missing entry.
func (state *State) sendHeartbeat() {
	Logger.Log(Logger.INFO, "Sending Heartbeat...")
	envelope := &Raft.Raft{}
	innerEnvelope := &Raft.Raft_AppendEntriesRequest{AppendEntriesRequest: nil}
	envelope.Message = innerEnvelope

	for i, server := range state.Servers {
		// Reset the struct for each server
		message := &Raft.AppendEntriesRequest{
			Term:         state.CurrentTerm,
			LeaderCommit: state.CommitIndex,
			LeaderId:     state.MyName,
			PrevLogIndex: 0,
			PrevLogTerm:  0,
			Entries:      []*Raft.LogEntry{},
		}

		// I don't want to send a heartbeat to myself
		if state.MyName == server.String() {
			continue
		}

		message.PrevLogIndex = state.NextIndex[i] - 1

		// In the Raft message array start at 1, but the log array starts at 0,
		// we then need to subtract 1 from the index to get the correct entry
		if uint64(len(state.Log)) > message.PrevLogIndex-1 {
			message.PrevLogTerm = state.Log[message.PrevLogIndex-1].Term
		}

		// If the server is missing a log entry, add them to the message
		if uint64(len(state.Log)) > state.NextIndex[i]-1 {
			message.Entries = append(message.Entries, state.Log[state.NextIndex[i]-1])
		}

		innerEnvelope.AppendEntriesRequest = message

		state.sendTo(server, envelope)
	}
}

// Server is the main loop of the server. It will listen for commands and requests.
// And handle them accordingly.
func (state *State) Server() {
	Logger.Log(Logger.INFO, "Server listening...")

	failures := 0
	var deadline time.Duration
	msg := &Raft.Raft{}

	for {
		// If the server command that will have an effect on it state it will change the state before handling requests

		// Check if a command has been sent through the CLI
		select {
		case cmd := <-command:
			// I have received a command handel it before reading requests
			state.commandsHandler(cmd)
			hasFinishedExecutingCommand <- true
		default:
			// If I have not received command read requests
		}

		if state.state == Follower {
			// I am a follower, I need to add random daley to my time
			// so that the leader election is not triggered at the same time for all the servers
			randomDelay := time.Duration(rand.Int31n(RandomTimeoutMax-RandomTimeoutMin)+RandomTimeoutMin) * time.Millisecond
			deadline = ElectionTimeout + randomDelay
		} else {
			deadline = BroadcastTime
		}

		// Check if the server has received a request
		address, err := Utils.ReadFromUDPConn(state.Listener, msg, deadline)

		if err != nil {
			if os.IsTimeout(err) {
				// The server has not received a request, before the set deadline
				switch state.state {
				case Follower:
					// I have not received a heartbeat; before the set deadline,
					// there seems to be no leader, I am going to start a leader election
					state.startLeaderElection()
				case Leader:
					// I need to send a heartbeat to all the other servers
					// Upon election: send initial empty AppendEntries RPCs
					// (heartbeat) to each server; repeat during idle periods to prevent election timeouts (§5.2)
					state.sendHeartbeat()
				case Candidate:
					// I have not received a vote from the majority of the servers before the set deadline
					state.resendRequestVoteMessage()
				}
			} else {
				Logger.Log(Logger.WARNING, "Failed to read from UDP listener")

				// We allow server to fail 3 times before stopping them
				// This is to avoid the server to stop if the network is not stable
				failures++

				if failures == 3 {
					Logger.Log(Logger.ERROR, "Too many failures stopping!")
					panic("Too many failures stopping!")
				}
			}

			// This is not a valid request, continue to the next iteration
			continue
		}
		// If the server has received a valid request, reset the number of failures
		failures = 0

		Logger.Log(Logger.INFO, "Message from address: "+address.String())

		state.messagesHandler(msg, address)
	}
}

// sendTo sends a message to a specific server.
// The message is Raft struct and will be marshalled before being sent.
func (state *State) sendTo(addr *net.UDPAddr, message *Raft.Raft) {

	err := Utils.WriteToUDPConn(state.Listener, addr, message)

	if err != nil {
		Logger.Log(Logger.WARNING, "Failed to send to host")
	}
}

// sendToAll sends a message to all the other servers.
func (state *State) sendToAll(message *Raft.Raft) {
	for _, addr := range state.Servers {
		if addr.String() == state.MyName {
			continue
		} // Don't send to self

		err := Utils.WriteToUDPConn(state.Listener, addr, message)

		if err != nil {
			Logger.Log(Logger.WARNING, "Failed to send to host")
		}
	}
}

// resendRequestVoteMessage is used to resend the request vote message to the servers that have not voted yet.
func (state *State) resendRequestVoteMessage() {

	Logger.Log(Logger.INFO, "Resending request vote message...")

	lastLogIndex := len(state.Log)

	lastLogTerm := uint64(0)
	if lastLogIndex > 0 {
		lastLogTerm = state.Log[lastLogIndex-1].Term
	}

	message := &Raft.Raft{Message: &Raft.Raft_RequestVoteRequest{RequestVoteRequest: &Raft.RequestVoteRequest{
		Term:          state.CurrentTerm,
		CandidateName: state.MyName,
		LastLogIndex:  uint64(lastLogIndex),
		LastLogTerm:   lastLogTerm,
	},
	}}

	for _, addr := range state.leftToVote {
		state.sendTo(addr, message)
	}
}

// startLeaderElection is used to start a leader election.
// It will set the state to candidate and send a request vote message to all the other servers.
func (state *State) startLeaderElection() {
	Logger.Log(Logger.INFO, "Starting leader election...")

	// Every time we start a new election, we reset the server that has not voted yet
	state.leftToVote = nil
	for _, server := range state.Servers {
		// I have voted for me, so remove me
		if server.String() == state.MyName {
			continue
		}

		state.leftToVote = append(state.leftToVote, server)
	}

	state.CurrentTerm++
	state.state = Candidate
	state.VotedFor = state.MyAddress
	state.votes = 1

	lastLogIndex := len(state.Log)

	lastLogTerm := uint64(0)
	if lastLogIndex > 0 {
		lastLogTerm = state.Log[lastLogIndex-1].Term
	}

	message := &Raft.Raft{Message: &Raft.Raft_RequestVoteRequest{RequestVoteRequest: &Raft.RequestVoteRequest{
		Term:          state.CurrentTerm,
		CandidateName: state.MyName,
		LastLogIndex:  uint64(lastLogIndex),
		LastLogTerm:   lastLogTerm,
	},
	}}

	Logger.Log(Logger.INFO, "Sending request vote to all other servers")

	state.sendToAll(message)
}

// Repl is an interface for the user to interact with the server through the commandline.
// It will send the commandString to the commandString channel and wait for the commandString to be executed.
func (state *State) repl() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nCommand> ")
		cmd, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			break
		}

		cmd = strings.TrimSpace(cmd)

		command <- cmd

		if <-hasFinishedExecutingCommand {
		}
	}
}

// loggerInit is used to initialize the logger.
func (state *State) loggerInit() {
	filename := "./" + "server-" + strings.Replace(state.MyName, ":", "-", 1) + ".log"

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to open log file!")
	}

	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func Start() {
	state := &State{
		CurrentTerm: 0,
		VotedFor:    nil,
		Log:         make([]*Entry, 0),
		CommitIndex: 0,
		LastApplied: 0,
		NextIndex:   nil,
		MatchIndex:  nil,
		Servers:     make([]*net.UDPAddr, 0),
		MyName:      "",
		state:       Follower,
	}

	state.Init()
	state.loggerInit()
	go state.repl()

	state.Server()

}
