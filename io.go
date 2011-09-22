package main

import (
	irc "cbeck/ircbot"
	mc "cbeck/mcserver"
	"bufio"
	"fmt"
	"os"
	"strings"
	)


var (
	chatRegex regex.Regexp 
	sanitizeRegex regex.Regexp
	commands chan *command
	commandResponse chan string
)

const (
	SOURCE_MC = iota
	SOURCE_IRC
)

func init() {
	chatRegex = regexp.MustCompile(`\[INFO\] (\* [a-zA-Z0-9\-]+|<[a-zA-Z0-9\-]>) (.*)`)
	sanitizeRegex = regexp.MustCompile("\n\r")
	commands = make(chan string, 1024)
	commandResponse = make(chan string, 1024)
}


func teeServerOutput() {
	var line string

	for {
		//The MC Server uses Stderr for almost, but not quite, everything.
		//Monitor both
		select {
		case line = <-server.Out:
		case line = <-server.Err:
		}

		//And dispatch to:

		fmt.Println(line) //The server console
		
		if matches := chatRegex.FindStringSubmatch(line); match != nil { //Irc, if it looks like chat
			if len(match) < 2 {
				continue
			}
			
			if match[1][0] == config.AttnChar { //Command issued from inside server
				commands <- &command{match[1][1:], SOURCE_MC}
			} else { //Chat
				bot.Send(&ircbot.Message{
				Command : "PRIVMSG",
				Args : []string{bot.IrcChan},
				Trailing : match[0] + match[1]
				})		
			}
		} 

		commandResponse <- line //The server output queue
	}	
}

func readConsoleInput() {
	in := bufio.NewReader(os.Stdin)

	for {
		line, _, err := in.ReadLine()
		if err != nil {
			logErr.Println(err)
			continue
		} else if len(line) < 1 {
			continue
		}

		server.In <- string(line)
	}
}

func echoIRCToServer(_ string, m ircbot.Message) string {
	sanitized := sanitize.Regex.ReplaceAllString(m.Trailing, " ")

	if m.Ctcp == "" { //Line was normal chat
		serverIn <- fmt.Sprintf("say <%s> %s", m.GetSender(), m.Trailing)
	} else if m.Ctcp == "ACTION"{ //Line was a Ctcp req
		serverIn <- fmt.Sprintf("say * %s %s", m.GetSender(), m.Trailing)
	} //Else ignore

	return ""
}