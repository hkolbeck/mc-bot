package main

import (
	re "regexp"
	"os"
	)



var commands chan<- *command

type command func([]string, string, <-chan string, chan<- string)

type call {
	command
	args []string 
	from string
}

func (server *Server) commandDispatch() {
	for cmd := range commands {
		cmd.command(cmd.args, cmd.from, serverOut, serverIn)
	}
}

var commandMap map[string]command = 
	map[string]command {
	"?" : helpCmd,
	"backup" : backupCmd,
	"ban" : banCmd,
	"give" : giveCmd,
	"help" : helpCmd,
	"kick" : kickCmd,
	"list" : listCmd,
	"mapgen" : mapgenCmd,
	"restart" : restartCmd,
	"source" : sourceCmd,
	"start" : startCmd,
	"state" : stateCmd,
	"stop" : stopCmd,
	"tp" : tpCmd,
}