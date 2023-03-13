package Server

import (
	"CADP-Project-3/Logger"
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"net"
)

func (state *State) commandMessageHandler(command string) {

}

func (state *State) requestVoteMessageHandler(request *Raft.RequestVoteRequest, address *net.UDPAddr) {
	Logger.Log(Logger.INFO, "Handling request vote message...")

	raftResponse := &Raft.RequestVoteResponse{}
	raftResponse.Term = state.CurrentTerm

	if state.CurrentTerm >= request.Term {
		// If the request term is less than the current term, then we reject the request
		raftResponse.VoteGranted = false

	} else if state.CurrentTerm == request.Term && state.state == Candidate {
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

		raftResponse.VoteGranted = true
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

	if state.votes < len(state.Servers)/2 {
		return
	}

	Logger.Log(Logger.INFO, "I am the leader!")

	state.state = Leader

	// Send an message to all other servers
	state.Send()
}

func (state *State) appendEntriesRequestMessageHandler(request *Raft.AppendEntriesRequest) {

}

func (state *State) appendEntriesResponseMessageHandler(response *Raft.AppendEntriesResponse) {

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
		state.appendEntriesRequestMessageHandler(v.AppendEntriesRequest)
	case *Raft.Raft_AppendEntriesResponse:
		state.appendEntriesResponseMessageHandler(v.AppendEntriesResponse)
	}
}
