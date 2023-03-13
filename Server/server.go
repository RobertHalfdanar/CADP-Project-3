package Server

import (
	"CADP-Project-3/Raft"
	"io"
	"log"
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

// This is the server loop
func (state *State) Server() {
	// TODO: Configure timeouts, read from the UDP connection, handle incoming messages and update state.
}

func (state *State) repl() {
	// TODO: Continuously read from stdin and handle the commands.
}

func main() {
	if true {
		log.SetOutput(io.Discard)
	}
	state := &State{
		CurrentTerm: 0,
		VotedFor:    nil,
		Log:         make([]*Entry, 0),
		CommitIndex: 0,
		LastApplied: 0,
		NextIndex:   nil,
		MatchIndex:  nil,
		Servers:     make([]*net.UDPAddr, 0),
		MyName:      os.Args[1],
		state:       Follower,
	}

	// TODO: Initialize the state, setup UDP listener and read the server list.

	go state.Server()
	state.repl()
}
