package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"fmt"
	"math"
	"net"
)

// forwardToLeader forwards the command to the leader
// Client can buy connected fo a follower, but the leader needs to receive the command.
func (state *State) forwardToLeader(command string) {
	if state.leader == nil {
		Logger.Log(Logger.INFO, "No leader, can't forward")
		return
	}

	message := &Raft.Raft{Message: &Raft.Raft_CommandName{CommandName: command}}
	state.sendTo(state.leader, message)

	Logger.Log(Logger.INFO, "Command forwarded to leader")
}

// commandMessageHandler handles a command message from the client
// If the server is a candidate, it will drop the command
// If the server is a follower, it will forward the command to the leader
func (state *State) commandMessageHandler(command string) {

	Logger.Log(Logger.INFO, "Handling command message...")
	if state.state == Candidate {
		Logger.Log(Logger.INFO, "Dropping command, I am candidate")
		return
	} else if state.state != Leader {
		Logger.Log(Logger.INFO, "Forwarding to leader...")
		state.forwardToLeader(command)
		return
	}

	newEntry := &Raft.LogEntry{CommandName: command, Term: state.CurrentTerm, Index: uint64(len(state.Log) + 1)}
	state.Log = append(state.Log, newEntry)

	index := Utils.Find(state.Servers, Utils.CreateUDPAddr(state.MyName))

	// Set my match index and next index
	state.MatchIndex[index] = state.NextIndex[index]
	state.NextIndex[index]++
}

// upToDate checks my log are up-to-date with another server log
func (state *State) upToDate(index uint64, term uint64) bool {
	if len(state.Log) == 0 {
		// My logs are empty, then I am not up-to-date
		return false
	}

	if state.Log[len(state.Log)-1].Term < term {
		// The other server last log term is higher the mine, I am not up-to-date
		return false
	} else if state.Log[len(state.Log)-1].Term > term {
		// The other server last log term is lower the mine, I am up-to-date
		return true
	}

	if uint64(len(state.Log)) > index {
		// I have more logs than the other server, I am up-to-date
		return true
	} else if uint64(len(state.Log)) < index {
		// I have fewer logs than the other server, I am not up-to-date
		return false
	}

	return false
}

// requestVoteMessageHandler handles a request vote message from another server
func (state *State) requestVoteMessageHandler(request *Raft.RequestVoteRequest, address *net.UDPAddr) {

	Logger.Log(Logger.INFO, "Handling request vote message from "+address.String())

	raftResponse := &Raft.RequestVoteResponse{}
	raftResponse.Term = state.CurrentTerm

	if request.Term > state.CurrentTerm {
		// The request term is greater than the current term, we are behind, and we need to be follower again
		// We might not vote for the candidate, but we need to update our term
		// If we don't vote for the candidate, we will time out and try to become a leader
		state.resetToFollower(request.Term)
	}

	if request.Term < state.CurrentTerm {
		// The request term is less than my current term, we reject the request
		raftResponse.VoteGranted = false

	} else if (state.VotedFor == nil || state.VotedFor.String() == address.String()) && !state.upToDate(request.LastLogIndex, request.LastLogTerm) {
		// Candidate is up-to-date, and we haven't voted yet, or we have voted for the candidate
		// We vote for the candidate
		state.CurrentTerm = request.Term
		state.state = Follower
		state.VotedFor = address
		state.MatchIndex = []uint64{}
		state.NextIndex = []uint64{}

		raftResponse.VoteGranted = true
	} else {
		raftResponse.VoteGranted = false
	}

	envelope := &Raft.Raft{}
	envelope.Message = &Raft.Raft_RequestVoteResponse{RequestVoteResponse: raftResponse}

	state.sendTo(address, envelope)

	if raftResponse.VoteGranted {
		Logger.Log(Logger.INFO, "Request vote handled with vote granted")
	} else {
		Logger.Log(Logger.INFO, "Request vote handled with vote not granted")
	}
}

// requestVoteResponseMessageHandler handles a request vote response message from another server
func (state *State) requestVoteResponseMessageHandler(response *Raft.RequestVoteResponse, address *net.UDPAddr) {
	Logger.Log(Logger.INFO, "Received vote response from: "+address.String())

	// Remove the server that has voted
	state.leftToVote = Utils.Remove(state.leftToVote, address)

	if response.VoteGranted {
		// The server has voted for us, we increment the number of votes we have
		state.votes++
	}

	majority := int(math.Floor(float64(len(state.Servers))/2.0)) + 1

	if state.votes < majority {
		// We don't have enough votes to become the leader
		Logger.Log(Logger.INFO, "I have "+fmt.Sprintf("%d", state.votes)+" votes, and I need "+fmt.Sprintf("%d", majority)+" votes to become the leader")
		return
	}

	// I have received enough votes to become the leader
	state.NextIndex = make([]uint64, len(state.Servers))
	state.MatchIndex = make([]uint64, len(state.Servers))

	myIndex := Utils.IndexOf(state.Servers, state.MyAddress)

	for i := range state.NextIndex {
		state.NextIndex[i] = uint64(len(state.Log) + 1)
		state.MatchIndex[i] = 0
	}
	state.MatchIndex[myIndex] = state.NextIndex[myIndex] - 1

	state.state = Leader

	Logger.Log(Logger.INFO, "I am the leader!, and I have "+fmt.Sprintf("%d", state.votes)+" votes")

	state.sendHeartbeat()
}

// appendEntriesFails sends a negative response to a AppendEntries request
// It a helper function to avoid code duplication
func (state *State) appendEntriesFails(address *net.UDPAddr) {
	appendEntriesResponse := &Raft.AppendEntriesResponse{}
	appendEntriesResponse.Term = state.CurrentTerm
	appendEntriesResponse.Success = false

	state.sendTo(
		address,
		&Raft.Raft{Message: &Raft.Raft_AppendEntriesResponse{AppendEntriesResponse: appendEntriesResponse}})
}

// newEntry adds a new entry to the log
func (state *State) newEntry(request *Raft.AppendEntriesRequest) {
	newEntry := request.Entries[0]

	// If an existing entry conflicts with a new one (same index but different terms),
	// delete the existing entry and all that follow it (§5.3)
	if newEntry.Index <= uint64(len(state.Log)) && state.Log[newEntry.Index-1].Term != newEntry.Term {
		state.Log[newEntry.Index-1] = newEntry
	} else {
		state.Log = append(state.Log, newEntry)
	}
}

// commitEntry moves the commit index to the last entry in the log
func (state *State) commitEntries(request *Raft.AppendEntriesRequest) {
	newEntry := &Entry{}
	if len(request.Entries) == 0 {
		newEntry.Index = math.MaxUint64
	} else {
		newEntry = request.Entries[0]
	}

	// set commitIndex = min(leaderCommit, index of last new entry)
	if request.LeaderCommit < newEntry.Index {
		// The new entry has not been committed
		state.CommitIndex = request.LeaderCommit
	} else {
		state.CommitIndex = newEntry.Index
	}
	state.commitEntry()
}

// appendEntriesRequestMessageHandler handles an append entries request message from another server
func (state *State) appendEntriesRequestMessageHandler(request *Raft.AppendEntriesRequest, address *net.UDPAddr) {

	if state.state == Leader {
		// If am a leader and I receive an appendEntriesRequest, I will become a follower,
		// there is another leader if I am on a lower term
		if request.Term > state.CurrentTerm {
			state.state = Follower
		} else {
			return
		}
	} else {
		// If I am not a leader, I will become a follower
		// This is the heartbeats
		state.state = Follower
	}

	state.leader = address

	if request.Term < state.CurrentTerm {
		// The request term is lower than the current term. I will not accept the request

		Logger.Log(Logger.INFO, "Append entries not successful, mismatching term")
		state.appendEntriesFails(address)
	} else if request.PrevLogIndex > 0 {
		if uint64(len(state.Log)) <= request.PrevLogIndex-1 {
			// I have fewer logs
			Logger.Log(Logger.INFO, "Append entries not successful, mismatching log index")
			state.appendEntriesFails(address)
			return
		} else if uint64(len(state.Log)) > request.PrevLogIndex-1 && state.Log[request.PrevLogIndex-1].Term != request.PrevLogTerm {
			// The log term of the prevLogIndex does not match
			Logger.Log(Logger.INFO, "Append entries not successful, entry has mismatching term")
			state.appendEntriesFails(address)
			return
		}
	}

	// The request has entries, I will add them to my log array
	if len(request.Entries) > 0 {
		state.newEntry(request)
	}

	// I will update the current term
	state.CurrentTerm = request.Term

	// If the leaderCommit is greater than the commitIndex, I will commit the entries
	if request.LeaderCommit > state.CommitIndex {
		state.commitEntries(request)
	}

	if len(request.Entries) > 0 {
		// I will send a positive response to the leader I received the entries and added them to my log
		appendEntriesResponse := &Raft.AppendEntriesResponse{}
		appendEntriesResponse.Term = state.CurrentTerm
		appendEntriesResponse.Success = true
		state.sendTo(address, &Raft.Raft{Message: &Raft.Raft_AppendEntriesResponse{AppendEntriesResponse: appendEntriesResponse}})

		Logger.Log(Logger.INFO, "Append entries successful, send response to leader")
	} else {
		Logger.Log(Logger.INFO, "Received heartbeat")
	}
}

// appendEntriesResponseMessageHandler handles an append entries response message from another server
func (state *State) appendEntriesResponseMessageHandler(response *Raft.AppendEntriesResponse, address *net.UDPAddr) {
	Logger.Log(Logger.INFO, "Handling entries response...")

	addressIndex := Utils.IndexOf(state.Servers, address)
	if addressIndex == -1 {
		panic("Should not happen")
	}

	if !response.Success {
		// The appendEntriesRequest was not successful, I will decrement the nextIndex and try again
		// The sever is missing logs
		state.NextIndex[addressIndex]--
		return
	}

	// The request was successful, I will update the matchIndex and nextIndex
	state.MatchIndex[addressIndex] = state.NextIndex[addressIndex]
	state.NextIndex[addressIndex]++

	// Count the number of servers that have the same matchIndex
	countIndex := map[uint64]int{}
	for _, matchIndex := range state.MatchIndex {

		count, ok := countIndex[matchIndex]

		if !ok {
			countIndex[matchIndex] = 1
		} else {
			countIndex[matchIndex] = count + 1
		}
	}

	leaderCommitIndex := state.CommitIndex

	majority := int(math.Floor(float64(len(state.Servers))/2.0)) + 1

	for k, v := range countIndex {
		if v >= majority && k > leaderCommitIndex {
			// The majority of servers have the same matchIndex, It is the new commit log index
			state.CommitIndex = k
			state.commitEntry()
			break
		}
	}
}

// messagesHandler handles the given messages
func (state *State) messagesHandler(raft *Raft.Raft, address *net.UDPAddr) {

	switch v := raft.Message.(type) {
	case *Raft.Raft_CommandName:
		state.commandMessageHandler(v.CommandName)
	case *Raft.Raft_RequestVoteRequest:
		if state.state == Failed {
			// Server is failed, I will not accept this message
			break
		}
		state.requestVoteMessageHandler(v.RequestVoteRequest, address)
	case *Raft.Raft_RequestVoteResponse:
		if state.state == Failed {
			// Server is failed, I will not accept this message
			break
		}
		state.requestVoteResponseMessageHandler(v.RequestVoteResponse, address)
	case *Raft.Raft_AppendEntriesRequest:
		if state.state == Failed {
			// Server is failed, I will not accept this message
			return
		}
		state.appendEntriesRequestMessageHandler(v.AppendEntriesRequest, address)
	case *Raft.Raft_AppendEntriesResponse:
		if state.state != Leader {
			return
		}
		state.appendEntriesResponseMessageHandler(v.AppendEntriesResponse, address)
	default:
		Logger.Log(Logger.ERROR, "Unknown message type")
	}
}
