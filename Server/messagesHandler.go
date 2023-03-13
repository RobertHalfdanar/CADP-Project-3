package Server

import "CADP-Project-3/Raft"

func commandMessageHandler(command string) {

}

func requestVoteMessageHandler(request *Raft.RequestVoteRequest) {

}

func requestVoteResponseMessageHandler(response *Raft.RequestVoteResponse) {

}

func appendEntriesRequestMessageHandler(request *Raft.AppendEntriesRequest) {

}

func appendEntriesResponseMessageHandler(response *Raft.AppendEntriesResponse) {

}

func messagesHandler(raft *Raft.Raft) {
	switch v := raft.Message.(type) {
	case *Raft.Raft_CommandName:
		commandMessageHandler(v.CommandName)
	case *Raft.Raft_RequestVoteRequest:
		requestVoteMessageHandler(v.RequestVoteRequest)
	case *Raft.Raft_RequestVoteResponse:
		requestVoteResponseMessageHandler(v.RequestVoteResponse)
	case *Raft.Raft_AppendEntriesRequest:
		appendEntriesRequestMessageHandler(v.AppendEntriesRequest)
	case *Raft.Raft_AppendEntriesResponse:
		appendEntriesResponseMessageHandler(v.AppendEntriesResponse)
	}
}
