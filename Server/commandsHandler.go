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

	dataColumnLength := 16

	if len(state.Servers)*3 > 16 && state.state == Leader {
		dataColumnLength = len(state.Servers)*2 + 4
	}

	format := "│ %-16s │ %-" + fmt.Sprintf("%d", dataColumnLength) + "s │\n"

	// Change this print to a table
	columns := []string{"My address", "Current term", "Voted for", "State", "Commit index", "Last applied", "Next index", "Match index"}
	data := []string{
		fmt.Sprintf("%s", state.MyName),
		fmt.Sprintf("%d", state.CurrentTerm),
		fmt.Sprintf("%s", state.VotedFor),
		fmt.Sprintf("%s", state.state.String()),
		fmt.Sprintf("%d", state.CommitIndex),
		fmt.Sprintf("%d", state.LastApplied),
		fmt.Sprintf("%d", state.NextIndex),
		fmt.Sprintf("%d", state.MatchIndex),
	}

	fmt.Printf("\n┌──────────────────┬" + strings.Repeat("─", dataColumnLength+2) + "┐\n")
	for i := 0; i < len(columns); i++ {
		fmt.Printf(format, columns[i], data[i])
	}
	fmt.Printf("└──────────────────┴" + strings.Repeat("─", dataColumnLength+2) + "┘\n")
}

func (state *State) logCommandHandler() {

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

	if state.state != Failed {
		return
	}

	state.state = Follower

	fmt.Println("\033[32mThis server is resumed\033[0m")
}

func (state *State) suspendCommandHandler() {

	if state.state == Failed {
		return
	}

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
