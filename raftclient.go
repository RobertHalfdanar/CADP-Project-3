package main

import (
	"CADP-Project-3/Client"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("\u001B[31mMissing server address\u001B[0m")
		fmt.Println("Usage: go run raftclient.go server-host:server-port")
	}

	serverAddress := os.Args[1]

	Client.Start(serverAddress)
}
