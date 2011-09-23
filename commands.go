package main

import (
	"strings"
	irc "cbeck/ircbot"
	)

type commandFunc func([]string) []string
type command struct {
	raw string
	source int
}

var commandMap map[string]commandFunc = 
	map[string]commandFunc {
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

func commandDispatch() {
	var reply []string

	for cmd := range commands {
		split := strings.Split(cmd.raw, " ")
		if len(split) < 1 {
			continue
		}

		f, ok := commandMap[split[0]]

		if !ok {
			reply = []string{"Unknown command: " + split[0]}
		} else {
			reply = f(split[1:])
		}

		switch cmd.source {
		case SOURCE_MC:
			for _, s := range reply {
				server.In <- "say " + s
			}
		case SOURCE_IRC:
			for _, s := range reply {
				bot.Send(&irc.Message{
				Command : "PRIVMSG",
				Args : []string{config.IrcChan},
				Trailing : s,
				})		
			}
		}
	}
}

func helpCmd(args []string) []string {
	return nil
}

func backupCmd(args []string) []string {
	return nil 
}

func banCmd(args []string) []string {
	return nil
}

func giveCmd(args []string) []string {
	return nil 
}

func kickCmd(args []string) []string {
	return nil 
}

func listCmd(args []string) []string {
	return nil 
}

func mapgenCmd(args []string) []string {
	return nil 
}

func restartCmd(args []string) []string {
	return nil 
}

func sourceCmd(args []string) []string {
	return nil 
}

func startCmd(args []string) []string {
	return nil 
}

func stateCmd(args []string) []string {
	return nil 
}

func stopCmd(args []string) []string {
	return nil 
}

func tpCmd(args []string) []string {
	return nil 
}
