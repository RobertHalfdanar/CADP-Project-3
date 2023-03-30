# RAFT done in GO

This README provides instructions on how to execute the project on your local machine.
## Prerequisites

In order to execute the project, you will need to have Go installed on your machine. If you do not already have Go installed, you can download it from the official website: https://golang.org/dl/

**Execution Steps**

Once you have Go installed, follow these steps to execute it:

    Open a terminal or command prompt.
    Navigate to the directory containing the Go program you want to execute.
    Build the program and run the file by running the following command:

    go run raftserver.go <MY IP ADDRESS>:<MY PORT> <PATH TO CONFIG FILE>

    Then to run a client run:
    
    go run raftclient.go IP:PORT

    where IP is the ip address of the server and PORT is the port of the server is listening on

**Running Orchestrator**

    <RÓBERT INSERT EXECUTING STEPS HERE>

**Notes**

The orchestrator will write the outputs of his servers <br>
and clients into the folder named orchestration

There are more detailed logs of each server and client in the Logs folder.<br>
They are however not named so knowing which log file is for which server is difficult

The ```.log``` files are located in the same directory as the ```.go``` programs
