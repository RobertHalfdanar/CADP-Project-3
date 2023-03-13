package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"bufio"
	"net"
	"os"
	"time"
)

const (
	BroadcastTime   = 20 * time.Millisecond
	ElectionTimeout = 20 * BroadcastTime
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
	lastAppendRequest []*Raft.AppendEntriesRequest // Did we get a response?
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

	fileName := os.Args[2]
	state.setServers(fileName)

	var err error
	state.Listener, err = Utils.CreateUDPListener(state.MyName)

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to create a udp listener!")
		os.Exit(-1)
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

		Logger.Log(Logger.INFO, "Message from address: "+address.String())
		// Logger.LogWithHost(Logger.INFO, address, "Received message")
	}
}

func (state *State) repl() {
	// TODO: Continuously read from stdin and handle the commands.
	Logger.Log(Logger.INFO, "Server sending...")

	for {
		time.Sleep(5 * time.Second)

		for _, addr := range state.Servers {
			if addr.String() == state.MyName {
				continue
			} // Don't send to self
			msg := &Raft.Raft{Message: nil}
			err := Utils.WriteToUDPConn(state.Listener, addr, msg)

			if err != nil {
				Logger.Log(Logger.WARNING, "Failed to send to host")
			}
		}
	}
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
	go state.Server()
	state.repl()
}
