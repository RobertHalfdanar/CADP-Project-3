package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	BroadcastTime   = 5000 * time.Millisecond
	ElectionTimeout = 2 * BroadcastTime // 20 * BroadcastTime
)

// States of the Server, cf figure 4
type RaftState int

const (
	Follower RaftState = iota
	Candidate
	Leader
	Failed
)

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
	MyAddress *net.UDPAddr   // Address we listen on
	Listener  *net.UDPConn   // Connection we listen on
	Servers   []*net.UDPAddr // List of all servers

	votes             int                          // Did the server get votes?
	leftToVote        []*net.UDPAddr               // How has voted
	lastAppendRequest []*Raft.AppendEntriesRequest // Did we get a response?

	leader *net.UDPAddr

	lock sync.RWMutex
}

func (state *State) getState() RaftState {
	state.lock.RLock()
	defer state.lock.RUnlock()

	return state.state
}

type Timer struct {
	timer time.Duration
	lock  sync.Mutex
}

var timer Timer

func (timer *Timer) increaseTimer(amount time.Duration) {
	timer.lock.Lock()
	defer timer.lock.Unlock()

	timer.timer += amount
}

func (timer *Timer) resetTimer() {
	timer.lock.Lock()
	defer timer.lock.Unlock()

	timer.timer = 0
}

func (timer *Timer) getTimer() time.Duration {
	timer.lock.Lock()
	defer timer.lock.Unlock()

	return timer.timer
}

func (state *State) setServers(fileName string) {
	file, err := os.OpenFile("./"+fileName, os.O_RDONLY, 0644)
	defer file.Close()

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to locate config file!")
		os.Exit(-1)
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
		os.Exit(-1)
	}
}

func (state *State) Init() {
	// TODO: Initialize the state, setup UDP listener and read the server list.
	Logger.Log(Logger.INFO, "Initialising state of server...")
	defer Logger.Log(Logger.INFO, "Server initialized")

	if len(os.Args) < 3 {
		Logger.Log(Logger.ERROR, "Too few arguments")
		os.Exit(-1)
	} else if len(os.Args) > 3 {
		Logger.Log(Logger.ERROR, "Too few arguments")
		os.Exit(-1)
	}

	state.MyName = os.Args[1]
	state.MyAddress = Utils.CreateUDPAddr(state.MyName)

	fileName := os.Args[2]
	state.setServers(fileName)

	var err error
	state.Listener, err = Utils.CreateUDPListener(state.MyName)
	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to create a udp listener!")
		os.Exit(-1)
	}
}

func (state *State) commitEntry() {
	Logger.Log(Logger.INFO, "Committing entry...")

	// log commit index entry
	log.Println(fmt.Sprintf("%d,%d,%s", state.Log[state.CommitIndex-1].Term, state.Log[state.CommitIndex-1].Index, state.Log[state.CommitIndex-1].CommandName))
}

func (state *State) sendHeartbeat() {
	Logger.Log(Logger.INFO, "Sending Heartbeat...")
	envelope := &Raft.Raft{}
	innerEnvelope := &Raft.Raft_AppendEntriesRequest{AppendEntriesRequest: nil}
	envelope.Message = innerEnvelope

	for i, server := range state.Servers {
		message := &Raft.AppendEntriesRequest{
			Term:         state.CurrentTerm,
			LeaderCommit: state.CommitIndex,
			LeaderId:     state.MyName,
			PrevLogIndex: 0,
			PrevLogTerm:  0,
			Entries:      []*Raft.LogEntry{},
		}

		if state.MyName == server.String() {
			continue
		}

		message.PrevLogIndex = state.NextIndex[i] - 1

		if uint64(len(state.Log)) > message.PrevLogIndex-1 {
			message.PrevLogTerm = state.Log[message.PrevLogIndex-1].Term
		}

		//  2 - 1 = 1

		if uint64(len(state.Log)) > state.NextIndex[i]-1 {

			message.Entries = append(message.Entries, state.Log[state.NextIndex[i]-1])
		}

		innerEnvelope.AppendEntriesRequest = message

		state.sendTo(server, envelope)
	}
}

func (state *State) flush() {
	start := time.Now()
	for {
		elapsed := time.Since(start)
		timer.increaseTimer(elapsed)
		switch state.getState() {
		case Follower:
			if timer.getTimer() >= ElectionTimeout {
				state.startLeaderElection()
				timer.resetTimer()
			}
		case Leader:
			if timer.getTimer() >= BroadcastTime {
				state.sendHeartbeat()
				timer.resetTimer()
			}
		case Candidate:
			if timer.getTimer() >= BroadcastTime {
				state.resendRequestVoteMessage()

				timer.resetTimer()
			}
		}

		start = time.Now()
		time.Sleep(1 * time.Millisecond)
	}
}

// This is the server loop
func (state *State) Server() {

	// TODO: Configure timeouts, read from the UDP connection, handle incoming messages and update state.

	Logger.Log(Logger.INFO, "Server listening...")

	failures := 0
	for {
		msg := &Raft.Raft{}
		address, err := Utils.ReadFromUDPConn(state.Listener, msg)

		if err != nil {
			Logger.Log(Logger.WARNING, "Failed to read from UDP listener")
			failures++

			if failures == 3 {
				Logger.Log(Logger.ERROR, "Too many failures stopping!")
				os.Exit(-1)
			}

			continue
		}
		failures = 0

		/*
			if !Utils.Contains(state.Servers, address) {
				Logger.Log(Logger.WARNING, "Unknown address received!")
				continue
			}
		*/

		Logger.Log(Logger.INFO, "Message from address: "+address.String())
		state.messagesHandler(msg, address)

		// Logger.LogWithHost(Logger.INFO, address, "Received message")
	}
}

func (state *State) sendTo(addr *net.UDPAddr, message *Raft.Raft) {

	err := Utils.WriteToUDPConn(state.Listener, addr, message)

	if err != nil {
		Logger.Log(Logger.WARNING, "Failed to send to host")
	}
}

func (state *State) sendToAll(message *Raft.Raft) {
	state.lock.RLock()
	defer state.lock.RUnlock()

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

func (state *State) resendRequestVoteMessage() {

	Logger.Log(Logger.INFO, "Resending request vote message...")

	state.lock.RLock()
	defer state.lock.RUnlock()

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

func (state *State) startLeaderElection() {
	state.lock.Lock()

	Logger.Log(Logger.INFO, "Starting leader election...")

	// Every time we start a new election we reset the server that have not voted yet
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

	// Issue a request vote to all other servers
	message := &Raft.Raft{Message: &Raft.Raft_RequestVoteRequest{RequestVoteRequest: &Raft.RequestVoteRequest{
		Term:          state.CurrentTerm,
		CandidateName: state.MyName,
		LastLogIndex:  uint64(lastLogIndex),
		LastLogTerm:   lastLogTerm,
	},
	}}
	state.lock.Unlock()

	Logger.Log(Logger.INFO, "Sending request vote to all other servers")

	state.sendToAll(message)
}

func (state *State) repl() {
	// TODO: Continuously read from stdin and handle the commands.
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Command> ")
		cmd, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			break
		}

		// Remove leading and trailing whitespace
		cmd = strings.TrimSpace(cmd)

		state.commandsHandler(cmd)
	}
}

func (state *State) loggerInit() {
	filename := "./" + "server-" + strings.Replace(state.MyName, ":", "-", 1) + ".log"

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to open log file!")
	}

	log.SetOutput(file)
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
	go state.Server()
	go state.flush()
	state.repl()
}
