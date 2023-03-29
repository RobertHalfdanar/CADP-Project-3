package Logger

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type LogLevel string

// Const's that define the type of log
// Makes it easier to read and filter through them
const (
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARNING"
	ERROR   LogLevel = "ERROR"
)

// colorLogLevels are fancy colors when we are printing to the terminal
// Currently is disabled because of user input
var colorLogLevels = map[string]string{
	string(INFO):    "\u001B[34mINFO\u001B[0m",
	string(WARNING): "\u001B[33mWARNING\u001B[0m",
	string(ERROR):   "\u001B[31mERROR\u001B[0m",
}

// getFormattedDateTime gets the current date and returns it as a string in a special format
func getFormattedDateTime() string {
	dateTime := time.Now()
	stringDateTime := dateTime.Format("02-Jan-2006 15:04:05")
	stringDateTime = strings.Replace(stringDateTime, ":", "-", -1)
	return stringDateTime
}

// The file name for the log file the process uses
var fileName = getFormattedDateTime() + "---" + strconv.Itoa(rand.Int()) + ".txt"

// logFormatter formats a message to the format we are using to write to our log file
func logFormatter(message string, host string, level LogLevel, color bool) string {
	if !color {
		return string(level) + ";" + host + ";" + getFormattedDateTime() + ";" + message
	}

	return colorLogLevels[string(level)] + ";" + host + ";" + getFormattedDateTime() + ";" + message
}

// writeLog opens the log file and writes the log message to it
// If the folder Logs or the log file does not exist,
// then it creates those to.
func writeLog(formattedMessage string) {
	file, err := os.OpenFile("./Logs/"+fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer file.Close()

	if err != nil {
		if _, err := os.Stat("./Logs"); os.IsNotExist(err) {
			err := os.MkdirAll("./Logs", 0700)
			if err != nil {
				panic("Error creating log folder - " + err.Error())
			}

			writeLog(formattedMessage)
			return
		} else {
			panic("Error creating/opening log file! - " + err.Error())
		}
	}

	_, err = fmt.Fprintln(file, formattedMessage)

	if err != nil {
		panic("Error writing to file! - " + err.Error())
	}
}

// PrintLogs opens the log file and prints its entire contents to the terminal
func PrintLogs() {
	file, err := os.OpenFile("./Logs/"+fileName, os.O_RDONLY, 0644)
	defer file.Close()

	if err != nil {
		fmt.Println("Error opening log file!")
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}

func printLog(message string) {
	fmt.Println(message)
}

// Log is the main log function that should be used
// Takes a log level which can be INFO, WARNING, or ERROR and a message
// and writes it into the log file
func Log(level LogLevel, message string) {
	formattedMessage := logFormatter(message, "NULL", level, false)

	// Prints the log message to the terminal with colors for the levels
	// Currently disabled because we are getting inputs from the users
	// colorFormattedMessage := logFormatter(message, "NULL", level, true)
	// printLog(colorFormattedMessage)
	writeLog(formattedMessage)
}

// LogWithHost is the same as Log but uses an extra parameter host
// which it uses to display an address in the log file
func LogWithHost(level LogLevel, host string, message string) {
	formattedMessage := logFormatter(message, host, level, false)

	// colorFormattedMessage := logFormatter(message, host, level, true)
	// printLog(colorFormattedMessage)
	writeLog(formattedMessage)
}
