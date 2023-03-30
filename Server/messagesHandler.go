package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"fmt"
	"math"
	"net"
	"strconv"
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
	state.lock.Lock()
	defer state.lock.Unlock()

	Logger.Log(Logger.INFO, "Handling command message...")
	if state.state == Candidate {
		Logger.Log(Logger.INFO, "Dropping command, I am candidate")
		return
	} else if state.state == Follower {
		Logger.Log(Logger.INFO, "Forwarding to leader...")
		state.forwardToLeader(command)
		return
	}

	newEntry := &Raft.LogEntry{CommandName: command, Term: state.CurrentTerm, Index: uint64(len(state.Log) + 1)}
	state.Log = append(state.Log, newEntry)

	index := Utils.Find(state.Servers, Utils.CreateUDPAddr(state.MyName))
	state.NextIndex[index]++
	state.MatchIndex[index]++
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
		timer.resetTimer(true)

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
	defer state.lock.Unlock()

	if state.state != Candidate {
		return
	}

	Logger.Log(Logger.INFO, "Received vote response from: "+address.String())

	// Remove the server that has voted
	state.leftToVote = Utils.Remove(state.leftToVote, address)

	if response.VoteGranted {
		// If the server has voted for us, then we increment the number of votes
		state.votes++
	}

	if state.votes < int(math.Ceil(float64(len(state.Servers))/2.0)) {
		return
	}

	// I have received enough votes to become the leader

	state.NextIndex = make([]uint64, len(state.Servers))
	state.MatchIndex = make([]uint64, len(state.Servers))

	for i := range state.NextIndex {
		state.NextIndex[i] = uint64(len(state.Log) + 1)
		state.MatchIndex[i] = state.CommitIndex
	}

	Logger.Log(Logger.INFO, "I am the leader!, and I have "+fmt.Sprintf("%d", state.votes)+" votes")

	state.state = Leader

	// Send a message to all other servers
}

func (state *State) AppendEntriesFails(address *net.UDPAddr) {
	appendEntriesResponse := &Raft.AppendEntriesResponse{}
	appendEntriesResponse.Term = state.CurrentTerm

	Logger.Log(Logger.INFO, "Append entries not successful, send response to leader")

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

	prevCommitted := state.CommitIndex

	if request.LeaderCommit < newEntry.Index {
		// The new entry has not been committed
		state.CommitIndex = request.LeaderCommit
	} else {
		state.CommitIndex = newEntry.Index
	}

	// The leader has committed more entries than we have
	numberOfCommits := state.CommitIndex - prevCommitted

	for ; numberOfCommits > 0; numberOfCommits-- {
		// Commit entries until we reach the leader commit index
		state.commitEntry()
	}

	state.LastApplied = state.CommitIndex
}

func (state *State) appendEntriesRequestMessageHandler(request *Raft.AppendEntriesRequest, address *net.UDPAddr) {
	state.lock.Lock()
	defer state.lock.Unlock()

	if state.state == Leader {
		if request.Term > state.CurrentTerm {
			state.state = Follower
		} else {
			return
		}
	} else {
		state.state = Follower
	}

	state.leader = address

	timer.resetTimer(true)

	// if and return -> AppendEntriesFails
	// if There is an entry in the message
	// If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)

	if request.Term < state.CurrentTerm {
		Logger.Log(Logger.INFO, "Append entries not successful, mismatching term")
		state.AppendEntriesFails(address)
		return

	} else if uint64(len(state.Log)) > request.PrevLogIndex-1 && state.Log[request.PrevLogIndex-1].Term != request.PrevLogTerm {
		// TODO  Reply false if log doesn't contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
		Logger.Log(Logger.INFO, "Append entries not successful, entry has mismatching term")
		state.AppendEntriesFails(address)
		return

	} else if len(request.Entries) > 0 {
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
	state.lock.Lock()

	Logger.Log(Logger.INFO, "Handling entries response...")

	addressIndex := Utils.IndexOf(state.Servers, address)
	if addressIndex == -1 {
		panic("Should not happen")
	}

	if response.Success == false {
		state.NextIndex[addressIndex]--
		state.lock.Unlock()
		return
	}

	state.NextIndex[addressIndex]++
	state.MatchIndex[addressIndex]++

	// TODO: If there exists an N such that N > commitIndex, a majority
	//		 of matchIndex[i] ≥ N, and log[N].term == currentTerm:
	//       set commitIndex = N (§5.3, §5.4)

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

	majority := int(math.Ceil(float64(len(state.Servers)) / 2.0))

	for k, v := range countIndex {
		fmt.Println(v, ">=", majority, "&&", k, ">", leaderCommitIndex)
		//  3 >= 3 && 3 > 2
		if v >= majority && k > leaderCommitIndex {
			fmt.Println("Yo")
			state.CommitIndex = k
			state.LastApplied = k
			state.commitEntry()
			break
		}
	}
	state.lock.Unlock()
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
