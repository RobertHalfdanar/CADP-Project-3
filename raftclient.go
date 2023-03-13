package main

import (
	"CADP-Project-3/raft"
	"fmt"
	proto "google.golang.org/protobuf/proto"
	"log"
	"os"
)

func main() {
	server := os.Args[1]
	fmt.Println("Input server argument: ", server)
	// Read from stdin and send
	data, err := proto.Marshal(&Raft.Raft{Message: &Raft.Raft_CommandName{CommandName: "command"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Example marshalled command: ", data)
}
