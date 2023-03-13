package Client

import (
	"CADP-Project-3/Raft"
	"CADP-Project-3/Utils"
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

// Server is a struct that represents a server
type Server struct {
	Addr *net.UDPAddr
	IP   string
	Port int32
}

// Send sends a message to the server, the server is specified in the Server struct
// if the server is not available, it will print an error message.
func (node *Server) Send(msg *Raft.Raft) {
	conn, err := net.DialUDP("udp", nil, node.Addr)
	defer conn.Close()

	if err != nil {
		fmt.Println("\u001B[31mFailed to connect to host\u001B[0m")
		return
	}

	// Send the message to the server
	err = Utils.WriteToConn(conn, msg)

	if err != nil {
		fmt.Println("\u001B[31mFailed to send to host\u001B[0m")
		return
	}
}

// The server that we are connected to
var server = Server{}

// exitCommandHandler handles the exit command
func exitCommandHandler() {
	fmt.Println("exit command")
}

// sendCommandHandler handles all other commands and sends them to the server
func sendCommandHandler(cmd string) {
	message := &Raft.Raft{Message: &Raft.Raft_CommandName{CommandName: cmd}}

	server.Send(message)

	fmt.Println("Command sent: " + cmd)
}

// commandsHandler find the correct handler for a given command
func commandsHandler(cmd string) {

	cmd = strings.TrimSpace(cmd)

	if Utils.HasPunctuation(cmd) {
		fmt.Println("\u001B[31mCommand cannot contain punctuation\u001B[0m")
		return
	}

	switch cmd {
	case "exit":
		exitCommandHandler()
	default:
		sendCommandHandler(cmd)
	}
}

// commandsListener listens for commands from the user, when a command is received it will be process it.
func commandsListener() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Command> ")
		cmd, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			break
		}

		commandsHandler(cmd)
	}
}

// Start initializes the server connection for the client and starts the command listener
func Start(serverAddress string) {

	server.IP, server.Port = Utils.ToIPAndPort(serverAddress)
	server.Addr = Utils.CreateUDPAddr(serverAddress)

	commandsListener()
}
