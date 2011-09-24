package main

import (
	irc "cbeck/ircbot"
	"bufio"
	"fmt"
	"os"
	"regexp"
	)


var (
	chatRegex *regexp.Regexp 
	sanitizeRegex *regexp.Regexp
	commands chan *command
	commandResponse chan string
)

const (
	SOURCE_MC = iota
	SOURCE_IRC
)

func init() {
	chatRegex = regexp.MustCompile(`\[INFO\]( \* [a-zA-Z0-9\-]+| <[a-zA-Z0-9\-]> )(.*)`)
	sanitizeRegex = regexp.MustCompile("\n\r")
	commands = make(chan *command, 1024)
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
		
		if matches := chatRegex.FindStringSubmatch(line); matches != nil { //Irc, if it looks like chat
			if len(matches) < 2 {
				continue
			}
			
			if matches[1][0] == bot.Attention { //Command issued from inside server
				commands <- &command{matches[1][1:], SOURCE_MC}
			} else { //Chat
				bot.Send(&irc.Message{
				Command : "PRIVMSG",
				Args : []string{config.IrcChan},
				Trailing : matches[0] + matches[1],
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

func echoIRCToServer(_ string, m irc.Message) string {
	sanitized := sanitizeRegex.ReplaceAllString(m.Trailing, " ")

	if m.Ctcp == "" { //Line was normal chat
		server.In <- fmt.Sprintf("say <%s> %s", m.GetSender(), sanitized)
	} else if m.Ctcp == "ACTION"{ //Line was a Ctcp req
		server.In <- fmt.Sprintf("say * %s %s", m.GetSender(), sanitized)
	} //Else ignore

	return ""
}