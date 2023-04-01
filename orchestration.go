package main

import (
	"CADP-Project-3/Logger"
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Node struct {
	cmd         *exec.Cmd
	address     string
	pipeWriter  *io.WriteCloser
	isSuspended bool
}

func (node *Node) Start() {
	err := node.cmd.Start()

	if err != nil {
		panic(err)
	}
}

func (node *Node) sendCommand(command string) {

	_, err := io.Writer(*node.pipeWriter).Write([]byte(command + "\n"))
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
}

var servers []*Node
var clients []*Node

var suspendedServers []*Node
var maxSus = 1

func createNode(fileToExec string, address string, processFilepath string, args ...string) *Node {

	tmp := []string{"run", fileToExec, address}
	tmp = append(tmp, args...)

	cmd := exec.Command("go", tmp...)
	processFilepath = processFilepath + strings.Replace(address, ":", "-", 1) + ".txt"

	file, err := os.OpenFile(processFilepath, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}

	cmd.Stdout = file
	cmd.Stderr = os.Stderr
	cmd.WaitDelay = time.Millisecond

	stdin, err := cmd.StdinPipe()

	node := &Node{
		address:    address,
		cmd:        cmd,
		pipeWriter: &stdin,
	}

	return node
}

func cleanup() {

	var receivedCommands [][]string
	error := false

	for index, server := range servers {
		if index == 0 {
			// Read from file
			file, _ := os.OpenFile("./server-"+strings.Replace(server.address, ":", "-", 1)+".log", os.O_RDONLY, 0644)

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				row := scanner.Text()

				columns := strings.Split(row, " ")[2]

				receivedCommands = append(receivedCommands, strings.Split(columns, ","))
			}
		}

		file, _ := os.OpenFile("./server-"+strings.Replace(server.address, ":", "-", 1)+".log", os.O_RDONLY, 0644)

		scanner := bufio.NewScanner(file)

		rowNumber := 0
		for scanner.Scan() {
			row := scanner.Text()

			tmp := strings.Split(row, " ")[2]
			columns := strings.Split(tmp, ",")

			if rowNumber > len(receivedCommands)-1 {
				error = true
			}

			if columns[0] != receivedCommands[rowNumber][0] {
				error = true

			}
			if columns[1] != receivedCommands[rowNumber][1] {
				error = true

			}
			if columns[2] != receivedCommands[rowNumber][2] {
				error = true
			}

			rowNumber++
		}
	}
	if !error {
		fmt.Println("All commands are the same for the servers!")
	} else {
		fmt.Println("Commands are not the same!")
	}

	for index, receivedCommand := range receivedCommands {
		if receivedCommand[2] != commands[index] {
			fmt.Println(receivedCommand[2], commands[index])
			fmt.Printf("\"%s\" command was not recived ", commands[index])
		}
	}
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()

	// read config file
	fileName := "config.local.txt"
	nClients := 1

	file, err := os.OpenFile("./"+fileName, os.O_RDONLY, 0644)
	defer file.Close()

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to locate config file!")
		panic("Failed to locate config file!")
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

	//simulateSuspend()

	simulateOneSuspend()
	//simulation()
}

var commands []string

func initCommands() {
	fileName := "commands.txt"

	file, err := os.OpenFile("./"+fileName, os.O_RDONLY, 0644)
	defer file.Close()

	if err != nil {
		Logger.Log(Logger.ERROR, "Failed to locate config file!")
		panic("Failed to locate config file!")
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
			time.Sleep(1 * time.Second)
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

	// Give time for a leader to emerge
	time.Sleep(10 * time.Second)

	// Client in its own thread
	go sendToServer(clients[0])

	// Sending commands to servers
	for {

		time.Sleep(10 * time.Second)
		for _, server := range servers {

			fmt.Println("Sending print command to server: " + server.address)
			server.sendCommand("print")
			time.Sleep(500 * time.Millisecond)
		}
	}

	/*

		if len(suspendedServers) < maxSus && rand.Int31n(100) < 50 {
			server.sendCommand("suspend")
		} else if len(suspendedServers) >= maxSus && rand.Int31n(100) < 50 {
			server.sendCommand("resume")
		}
	*/

}

func simulateOneSuspend() {
	for _, server := range servers {
		server.Start()
	}

	for _, client := range clients {
		client.Start()
	}

	// Give time for a leader to emerge
	time.Sleep(10 * time.Second)

	// Client in its own thread
	go sendToServer(clients[0])

	// Sending commands to servers

	timeStart := time.Now()
	timer2 := time.Now()

	for {
		time.Sleep(200 * time.Millisecond)

		currentTime := time.Now()
		for _, server := range servers {
			server.sendCommand("print")
		}

		if currentTime.Sub(timer2).Seconds() > 80 {
			servers[3].sendCommand("resume")
			break
		}

		if currentTime.Sub(timeStart).Seconds() > 5 {
			servers[3].sendCommand("suspend")
			timeStart = time.Now()
		}
	}

	time.Sleep(10 * time.Second)

	for _, server := range servers {
		server.sendCommand("suspend")
		server.sendCommand("print")
	}

	time.Sleep(1 * time.Second)
}

func simulateSuspend() {
	for _, server := range servers {
		server.Start()
	}

	for _, client := range clients {
		client.Start()
	}

	// Give time for a leader to emerge
	time.Sleep(10 * time.Second)

	// Client in its own thread
	go sendToServer(clients[0])

	// Sending commands to servers

	timeStart := time.Now()
	timer2 := time.Now()

	for {
		time.Sleep(5 * time.Second)

		currentTime := time.Now()
		for _, server := range servers {
			server.sendCommand("print")
		}

		if currentTime.Sub(timer2).Seconds() > 5 {
			fmt.Println("Simulation ended!")
			for _, server := range servers {
				if server.isSuspended {
					server.sendCommand("resume")
				}
			}

			break
		}

		// Every 2 seconds suspend a random server
		if currentTime.Sub(timeStart).Seconds() > 2 {
			server := servers[rand.Int31n(int32(len(servers)))]

			if rand.Int31n(2) == 1 {

				if !server.isSuspended {
					continue
				}

				fmt.Println("Resuming server: " + server.address)
				server.sendCommand("resume")
				server.isSuspended = false

			} else {
				if server.isSuspended {
					continue
				}

				fmt.Println("Suspending server: " + server.address)

				server.sendCommand("suspend")
				server.isSuspended = true
			}

			timeStart = time.Now()
		}
	}

	time.Sleep(10 * time.Second)

	for _, server := range servers {
		server.sendCommand("print")
	}

	time.Sleep(1 * time.Second)
}
