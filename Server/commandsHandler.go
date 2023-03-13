package Server

import (
	"fmt"
)

/*
log
print
resume
suspend
*/

func printCommandHandler() {
	fmt.Println("print command")

}

func logCommandHandler() {
	fmt.Println("log command")
}

func resumeCommandHandler() {
	fmt.Println("resume command")
}

func suspendCommandHandler() {
	fmt.Println("suspend command")
}

// commandMapper maps a command to a function
var commandMapper = map[string]func(){
	"print":   printCommandHandler,
	"log":     logCommandHandler,
	"resume":  resumeCommandHandler,
	"suspend": suspendCommandHandler,
}

// commandsHandler find the correct handler for a given command
func commandsHandler(cmd string) {
	commandFunction, ok := commandMapper[cmd]

	if !ok {
		fmt.Printf("command not understood: %s\n", cmd)
		return
	}

	commandFunction()
}
