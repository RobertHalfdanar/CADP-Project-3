package Server

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

/*
log
print
resume
suspend
*/

func (state *State) printCommandHandler() {
	fmt.Println("print command")

	state.lock.RLock()
	defer state.lock.RUnlock()

	println()

	/*
		func printNodeRegistry(node *RegisteredNode) {
			if node.registry == nil {
				return
			}

			fmt.Printf("┌───── Node %-3d Registry ─────┐\n", node.id)
			fmt.Printf("│ %-3s │ %-21s │\n", "ID", "Address")
			fmt.Println("├─────┼───────────────────────┤")

			for _, node := range node.registry.Peers {
				formattedId := fmt.Sprintf("│ %3d │ %21s │", node.Id, node.Address)
				fmt.Println(formattedId)
			}

			fmt.Println("└─────┴───────────────────────┘")
		}
	*/

	// Change this print to a table

	fmt.Println("│ Current term: ", state.CurrentTerm)
	fmt.Println("│ Voted for: ", state.VotedFor)
	fmt.Println("│ State: ", state.state)
	fmt.Println("│ Commit index: ", state.CommitIndex)
	fmt.Println("│ Last applied: ", state.LastApplied)
	fmt.Println("│ Next index: ", state.NextIndex)
	fmt.Println("│ Match index: ", state.MatchIndex)
	println()
}

func (state *State) logCommandHandler() {
	fmt.Println("log command")

	filenamePath := "./" + "server-" + strings.Replace(state.MyName, ":", "-", 1) + ".log"

	file, err := os.OpenFile(filenamePath, os.O_RDONLY, 0644)
	defer file.Close()

	if err != nil {
		fmt.Println("Error opening log file!")
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}

func (state *State) resumeCommandHandler() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.state = Follower
	timer.resetTimer()

	fmt.Println("\033[32mThis server is resumed\033[0m")
}

func (state *State) suspendCommandHandler() {
	state.lock.Lock()
	defer state.lock.Unlock()

	state.state = Failed
	fmt.Println("\033[33mThis server is suspended\033[0m")
}

// commandsHandler find the correct handler for a given command
func (state *State) commandsHandler(cmd string) {

	switch cmd {
	case "print":
		state.printCommandHandler()
	case "log":
		state.logCommandHandler()
	case "resume":
		state.resumeCommandHandler()
	case "suspend":
		state.suspendCommandHandler()
	default:
		fmt.Println("Command not found")
	}
}
