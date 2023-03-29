package main

import (
	"CADP-Project-3/Logger"
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Node struct {
	cmd        *exec.Cmd
	address    string
	pipeWriter *io.PipeWriter
}

func (node *Node) Start() {
	err := node.cmd.Start()

	if err != nil {
		panic(err)
	}
}

func (node *Node) sendCommand(command string) {
	_, err := node.pipeWriter.Write([]byte(command + "\n"))

	if err != nil {
		panic(err)
	}
}

var servers []*Node
var clients []*Node

func createNode(fileToExec string, address string, processFilepath string, args ...string) *Node {
	r, w := io.Pipe()

	tmp := []string{"run", fileToExec, address}
	tmp = append(tmp, args...)

	cmd := exec.Command("go", tmp...)
	processFilepath = processFilepath + strings.Replace(address, ":", "-", 1) + ".txt"

	file, err := os.OpenFile(processFilepath, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}

	cmd.Stdin = r
	cmd.Stdout = file
	cmd.Stderr = os.Stderr
	cmd.WaitDelay = time.Millisecond

	node := &Node{
		address:    address,
		cmd:        cmd,
		pipeWriter: w,
	}

	return node
}

func main() {
	// read config file
	fileName := "config.local.txt"
	nClients := 1

	file, err := os.OpenFile("./"+fileName, os.O_RDONLY, 0644)
	defer file.Close()

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to locate config file!")
		os.Exit(-1)
	}

	if _, err := os.Stat("./orchestration"); os.IsNotExist(err) {
		os.MkdirAll("./orchestration", 0700) // Create your file
	}

	processPath := "./orchestration/"

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		address := scanner.Text()

		node := createNode("./raftserver.go", address, processPath+"server-", fileName)
		servers = append(servers, node)
	}

	for i := 0; i < nClients; i++ {
		randomServerIndex := rand.Int31n(int32(len(servers)))

		address := servers[randomServerIndex].address

		node := createNode("./raftclient.go", address, processPath+"client-")
		clients = append(clients, node)
	}

	initCommands()

	simulation()

}

var commands []string

func initCommands() {
	fileName := "commands.txt"

	file, err := os.OpenFile("./"+fileName, os.O_RDONLY, 0644)
	defer file.Close()

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to locate config file!")
		os.Exit(-1)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		command := scanner.Text()
		commands = append(commands, command)
	}
}

func sendToServer(client *Node) {
	for _, client := range clients {
		for _, command := range commands {
			client.sendCommand(command)
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func simulation() {
	for _, server := range servers {
		server.Start()
	}

	for _, client := range clients {
		client.Start()
	}

	time.Sleep(10 * time.Second)

	go sendToServer(clients[0])

	for {
		time.Sleep(20 * time.Second)
		for _, server := range servers {
			server.sendCommand("print")
			fmt.Println("Printed")

		}

	}

}
