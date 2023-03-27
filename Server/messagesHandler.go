package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"net"
	"strconv"
)

func (state *State) commandMessageHandler(command string) {

	state.lock.Lock()
	defer state.lock.Unlock()

	Logger.Log(Logger.INFO, "Handling command message...")

	newEntry := &Raft.LogEntry{CommandName: command, Term: state.CurrentTerm, Index: uint64(len(state.Log) - 1)}
	state.Log = append(state.Log, newEntry)
}

func (state *State) requestVoteMessageHandler(request *Raft.RequestVoteRequest, address *net.UDPAddr) {
	state.lock.Lock()

	Logger.Log(Logger.INFO, "Handling request vote message...")

	raftResponse := &Raft.RequestVoteResponse{}
	raftResponse.Term = state.CurrentTerm

	Logger.Log(Logger.INFO, strconv.Itoa(int(state.CurrentTerm)))
	Logger.Log(Logger.INFO, strconv.Itoa(int(request.Term)))

	if state.CurrentTerm > request.Term {
		// If the request term is less than the current term, then we reject the request
		raftResponse.VoteGranted = false

	} else if state.state == Candidate && state.CurrentTerm == request.Term {
		// If the request term is equal to the current term, and we are a candidate, then we reject the request
		// I vote for myself and can only vote for one candidate
		raftResponse.VoteGranted = false

	} else if state.LastApplied > request.LastLogIndex {
		// If our log index is large then the request log index, then we reject the request
		// I have more logs
		raftResponse.VoteGranted = false

	} else {
		// We vote for the candidate

		state.CurrentTerm = request.Term
		state.state = Follower
		state.VotedFor = address
		timer.resetTimer()

		raftResponse.VoteGranted = true
	}

	envelope := &Raft.Raft{}
	envelope.Message = &Raft.Raft_RequestVoteResponse{RequestVoteResponse: raftResponse}

	state.lock.Unlock()
	state.sendTo(address, envelope)

	if raftResponse.VoteGranted {
		Logger.Log(Logger.INFO, "Request vote handled with vote granted")
	} else {
		Logger.Log(Logger.INFO, "Request vote handled with vote not granted")
	}
}

func (state *State) requestVoteResponseMessageHandler(response *Raft.RequestVoteResponse, address *net.UDPAddr) {
	state.lock.Lock()

	Logger.Log(Logger.INFO, "Received vote response from: "+address.String())

	// Remove the server that has voted
	state.leftToVote = Utils.Remove(state.leftToVote, address)

	if response.VoteGranted {
		// If the server has voted for us, then we increment the number of votes
		state.votes++
	}

	if state.votes < len(state.Servers)/2 {
		return
	}

	state.NextIndex = make([]uint64, len(state.Servers))
	state.MatchIndex = make([]uint64, len(state.Servers))

	for i := range state.NextIndex {
		state.NextIndex[i] = uint64(len(state.Log) + 1)
		state.MatchIndex[i] = 0
	}

	Logger.Log(Logger.INFO, "I am the leader!")

	state.state = Leader

	// Send a message to all other servers
	state.lock.Unlock()
}

func (state *State) appendEntriesRequestMessageHandler(request *Raft.AppendEntriesRequest, address *net.UDPAddr) {

	appendEntriesResponse := &Raft.AppendEntriesResponse{}
	appendEntriesResponse.Term = state.CurrentTerm
	appendEntriesResponse.Success = true

	timer.resetTimer()

	if len(request.Entries) == 0 {
		Logger.Log(Logger.INFO, "Received Heartbeat")
		return

	} else if request.Term < state.CurrentTerm {
		// TODO 1. Reply false if term < currentTerm (§5.1)
		Logger.Log(Logger.INFO, "Append entries not successful, mismatching term")
		appendEntriesResponse.Success = false
	}

	if request.PrevLogIndex != 0 {
		term := state.Log[request.PrevLogIndex-1].Term

		if term != request.PrevLogTerm {
			// TODO  Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
			Logger.Log(Logger.INFO, "Append entries not successful, entry has mismatching term")
			appendEntriesResponse.Success = false
		}
	}

	if appendEntriesResponse.Success == false {
		Logger.Log(Logger.INFO, "Append entries not successful, send response to leader")

		state.sendTo(
			address,
			&Raft.Raft{Message: &Raft.Raft_AppendEntriesResponse{AppendEntriesResponse: appendEntriesResponse}})

		return
	}

	newEntry := request.Entries[0]
	newAllocatedEntry := new(Entry)
	*newAllocatedEntry = *newEntry

	// TODO If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it (§5.3)
	if newEntry.Index <= uint64(len(state.Log)) && state.Log[newEntry.Index].Term != newEntry.Term {
		state.Log[newEntry.Index] = newAllocatedEntry
	} else { // TODO Append any new entries not already in the log
		state.Log = append(state.Log, newEntry)
	}

	// TODO If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if request.LeaderCommit > state.CommitIndex {
		if request.LeaderCommit < newEntry.Index {
			state.CommitIndex = request.LeaderCommit
		} else {
			state.CommitIndex = newEntry.Index
		}

	}

	Logger.Log(Logger.INFO, "Append entries successful, send response to leader")
	state.sendTo(address, &Raft.Raft{Message: &Raft.Raft_AppendEntriesResponse{AppendEntriesResponse: appendEntriesResponse}})
}

func (state *State) appendEntriesResponseMessageHandler(response *Raft.AppendEntriesResponse, address *net.UDPAddr) {
	state.lock.Lock()
	defer state.lock.Unlock()

	addressIndex := Utils.IndexOf(state.Servers, address)
	if addressIndex == -1 {
		panic("Should not happen")
	}

	if response.Success == false {
		state.NextIndex[addressIndex]--
	} else {
		state.NextIndex[addressIndex]++
		state.MatchIndex[addressIndex]++
	}
}

func (state *State) messagesHandler(raft *Raft.Raft, address *net.UDPAddr) {
	switch v := raft.Message.(type) {
	case *Raft.Raft_CommandName:
		state.commandMessageHandler(v.CommandName)
	case *Raft.Raft_RequestVoteRequest:
		state.requestVoteMessageHandler(v.RequestVoteRequest, address)
	case *Raft.Raft_RequestVoteResponse:
		state.requestVoteResponseMessageHandler(v.RequestVoteResponse, address)
	case *Raft.Raft_AppendEntriesRequest:
		state.appendEntriesRequestMessageHandler(v.AppendEntriesRequest, address)
	case *Raft.Raft_AppendEntriesResponse:
		state.appendEntriesResponseMessageHandler(v.AppendEntriesResponse, address)
	}
}
