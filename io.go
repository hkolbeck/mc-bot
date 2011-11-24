package main

import (
	irc "cbeck/ircbot"
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"regexp"
	)


var (
	chatRegex *regexp.Regexp 
	sanitizeRegex *regexp.Regexp
	commands chan *command
	commandResponse chan string
	serverErrors int
	severeServerErrors int
	serverVersion string
)

const (
	SOURCE_MC = iota
	SOURCE_IRC
)

func init() {
	chatRegex = regexp.MustCompile(`\[INFO\]( \* [a-zA-Z0-9\-]+| <[a-zA-Z0-9\-]+> )(.*)`)
	sanitizeRegex = regexp.MustCompile("\n\r")
	commands = make(chan *command, 1024)
	commandResponse = make(chan string, 2048)
	go func() {
		for sig := range signal.Incoming {
			usig, _ := sig.(os.UnixSignal)
			fmt.Fprintln(os.Stderr, "Recieved signal: " + sig.String())
			switch (usig) {
			case syscall.SIGINT, syscall.SIGTERM:
				server.Destroy()
				os.Exit(1)
			case syscall.SIGHUP:
				err := config.Reparse()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Config reparse failed: " + err.String())
				}
			}
		}
	}()
}


func teeServerOutput() {
	var line string
	senderRegex := regexp.MustCompile(`\[INFO\] (\* |<)([a-zA-Z0-9\-]+)[> ]`)
	errorRegex := regexp.MustCompile(`java.*Exception`) 
	severeErrorRegex := regexp.MustCompile(`\[SEVERE\] Unexpected exception`) 

	for {
		//The MC Server uses Stderr for almost, but not quite, everything.
		//Monitor both
		select {
		case line = <-server.Out:
		case line = <-server.Err:
		}

		if errorRegex.MatchString(line) {
			serverErrors++
		} else if severeErrorRegex.MatchString(line) {
			severeServerErrors++
		}

		//And dispatch to:

		fmt.Println(line) //The server console
		
		if matches := chatRegex.FindStringSubmatch(line); matches != nil { //Irc, if it looks like chat
			if len(matches) < 3 {
				continue
			}
			
			if matches[2][0] == bot.Attention { //Command issued from inside server
				senderMatches := senderRegex.FindStringSubmatch(line)
				commands <- &command{matches[2][1:], senderMatches[2], SOURCE_MC}
				logInfo.Printf("%s sent command '%s' from in-server", senderMatches[2], matches[2][1:])
			} else { //Chat
				bot.Send(&irc.Message{
				Command : "PRIVMSG",
				Args : []string{config.IrcChan},
				Trailing : matches[1] + matches[2],
				})		
			}
		} 
		
		select {
		case commandResponse <- line: //The server output queue
		case <-commandResponse: //If the buffer has filled, drop the oldest line
			commandResponse <- line
		}
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

		//A 'stop' issued at the console could easily muck things up.
		//Hijack it.
		if string(line) == "stop" {
			server.Stop(1e9, "Stop issued at console. Going down now!")
			serverErrors = 0
			severeServerErrors = 0
		} else {
			server.In <- string(line)
		}		
	}
}

func echoIRCToServer(_ string, m *irc.Message) string {
	sanitized := sanitizeRegex.ReplaceAllString(m.Trailing, " ")

	if m.Ctcp == "" { //Line was normal chat
		server.In <- fmt.Sprintf("say <%s> %s", m.GetSender(), sanitized)
	} else if m.Ctcp == "ACTION"{ //Line was a Ctcp req
		server.In <- fmt.Sprintf("say * %s %s", m.GetSender(), sanitized)
	} //Else ignore

	return ""
}