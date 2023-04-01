package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"fmt"
	"math"
	"net"
)

func (state *State) forwardToLeader(command string) {
	if state.leader == nil {
		Logger.Log(Logger.INFO, "No leader, can't forward")
		return
	}

	message := &Raft.Raft{Message: &Raft.Raft_CommandName{CommandName: command}}
	state.sendTo(state.leader, message)

	Logger.Log(Logger.INFO, "Command forwarded to leader")
}

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
	state.MatchIndex[index] = state.NextIndex[index]
	state.NextIndex[index]++
}

func (state *State) upToDate(index uint64, term uint64) bool {
	if len(state.Log) == 0 {
		return true
	}

	if state.Log[len(state.Log)-1].Term < term {
		return true
	} else if state.Log[len(state.Log)-1].Term > term {
		return false
	}

	if uint64(len(state.Log)) > index {
		return false
	} else if uint64(len(state.Log)) < index {
		return true
	}

	return true
}

func (state *State) requestVoteMessageHandler(request *Raft.RequestVoteRequest, address *net.UDPAddr) {

	Logger.Log(Logger.INFO, "Handling request vote message from "+address.String())

	raftResponse := &Raft.RequestVoteResponse{}
	raftResponse.Term = state.CurrentTerm

	if request.Term < state.CurrentTerm {
		// If the request term is less than the current term, then we reject the request
		raftResponse.VoteGranted = false

	} else if (state.VotedFor == nil || state.VotedFor.String() == address.String()) && state.upToDate(request.LastLogIndex, request.LastLogTerm) {
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

	if request.Term > state.CurrentTerm {
		state.state = Follower
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

func (state *State) requestVoteResponseMessageHandler(response *Raft.RequestVoteResponse, address *net.UDPAddr) {
	Logger.Log(Logger.INFO, "Received vote response from: "+address.String())

	// Remove the server that has voted
	state.leftToVote = Utils.Remove(state.leftToVote, address)

	if response.VoteGranted {
		// If the server has voted for us, then we increment the number of votes
		state.votes++
	}

	majority := int(math.Floor(float64(len(state.Servers))/2.0)) + 1

	if state.votes < majority {
		return
	}

	// I have received enough votes to become the leader

	state.NextIndex = make([]uint64, len(state.Servers))
	state.MatchIndex = make([]uint64, len(state.Servers))

	for i := range state.NextIndex {
		state.NextIndex[i] = uint64(len(state.Log) + 1)
		state.MatchIndex[i] = 0
	}

	Logger.Log(Logger.INFO, "I am the leader!, and I have "+fmt.Sprintf("%d", state.votes)+" votes")

	state.state = Leader
	state.sendHeartbeat()
	// Send a message to all other servers
}

func (state *State) AppendEntriesFails(address *net.UDPAddr) {
	appendEntriesResponse := &Raft.AppendEntriesResponse{}
	appendEntriesResponse.Term = state.CurrentTerm
	appendEntriesResponse.Success = false

	state.sendTo(
		address,
		&Raft.Raft{Message: &Raft.Raft_AppendEntriesResponse{AppendEntriesResponse: appendEntriesResponse}})
}

func (state *State) newEntry(request *Raft.AppendEntriesRequest) {
	newEntry := request.Entries[0]
	newAllocatedEntry := new(Entry)
	*newAllocatedEntry = *newEntry

	// TODO If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it (§5.3)
	if newEntry.Index <= uint64(len(state.Log)) && state.Log[newEntry.Index-1].Term != newEntry.Term {
		state.Log[newEntry.Index-1] = newAllocatedEntry
	} else { // TODO Append any new entries not already in the log
		state.Log = append(state.Log, newEntry)
	}
}

func (state *State) commitEntries(request *Raft.AppendEntriesRequest) {
	// TODO If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	newEntry := &Entry{}
	if len(request.Entries) == 0 {
		newEntry.Index = math.MaxUint64
	} else {
		newEntry = request.Entries[0]
	}

	if request.LeaderCommit < newEntry.Index {
		// The new entry has not been committed
		state.CommitIndex = request.LeaderCommit
	} else {
		state.CommitIndex = newEntry.Index
	}

	state.commitEntry()
}

func (state *State) appendEntriesRequestMessageHandler(request *Raft.AppendEntriesRequest, address *net.UDPAddr) {
	if state.state == Leader {
		if request.Term > state.CurrentTerm {
			state.state = Follower
		} else {
			return
		}
	} else {
		state.state = Follower
	}

	state.VotedFor = nil
	state.leader = address

	// if and return -> AppendEntriesFails
	// if There is an entry in the message
	// If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if request.Term < state.CurrentTerm {
		Logger.Log(Logger.INFO, "Append entries not successful, mismatching term")
		state.AppendEntriesFails(address)
		return
		//    true 										0 < 0 false ||
	} else if request.PrevLogIndex > 0 {
		if uint64(len(state.Log)) <= request.PrevLogIndex-1 {
			Logger.Log(Logger.INFO, "Append entries not successful, mismatching log index")
			state.AppendEntriesFails(address)
			return
		} else if uint64(len(state.Log)) > request.PrevLogIndex-1 && state.Log[request.PrevLogIndex-1].Term != request.PrevLogTerm {
			Logger.Log(Logger.INFO, "Append entries not successful, entry has mismatching term")
			state.AppendEntriesFails(address)
			return
		}
	}

	if len(request.Entries) > 0 {
		state.newEntry(request)
	}

	state.CurrentTerm = request.Term

	if request.LeaderCommit > state.CommitIndex {
		state.commitEntries(request)
	}

	if len(request.Entries) > 0 {
		appendEntriesResponse := &Raft.AppendEntriesResponse{}
		appendEntriesResponse.Term = state.CurrentTerm
		appendEntriesResponse.Success = true
		state.sendTo(address, &Raft.Raft{Message: &Raft.Raft_AppendEntriesResponse{AppendEntriesResponse: appendEntriesResponse}})

		Logger.Log(Logger.INFO, "Append entries successful, send response to leader")
	} else {
		Logger.Log(Logger.INFO, "Received heartbeat")
	}
}

func (state *State) appendEntriesResponseMessageHandler(response *Raft.AppendEntriesResponse, address *net.UDPAddr) {
	// If new leader was elected, ignore response I am leader no more

	Logger.Log(Logger.INFO, "Handling entries response...")

	addressIndex := Utils.IndexOf(state.Servers, address)
	if addressIndex == -1 {
		panic("Should not happen")
	}

	if response.Success == false {
		if response.Term > state.CurrentTerm {
			state.state = Follower
			return
		}

		state.NextIndex[addressIndex]--
		return
	}

	state.MatchIndex[addressIndex] = state.NextIndex[addressIndex]
	state.NextIndex[addressIndex]++

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
			state.CommitIndex = k
			state.commitEntry()
			break
		}
	}
}

func (state *State) messagesHandler(raft *Raft.Raft, address *net.UDPAddr) {

	switch v := raft.Message.(type) {
	case *Raft.Raft_CommandName:
		state.commandMessageHandler(v.CommandName)
	case *Raft.Raft_RequestVoteRequest:
		if state.state == Failed {
			break
		}
		state.requestVoteMessageHandler(v.RequestVoteRequest, address)
	case *Raft.Raft_RequestVoteResponse:
		if state.state == Failed {
			break
		}
		state.requestVoteResponseMessageHandler(v.RequestVoteResponse, address)
	case *Raft.Raft_AppendEntriesRequest:
		if state.state == Failed {
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
