package main

import (
	"cbeck/ircbot"
	"bufio"
	"fmt"
	"os"
	"strings"
	)


var (
	commandQueue chan<- command
	serverIn chan<- string
	serverOut <-chan string
)

func readServerOutput(server Server, out chan<- string) {
	src, err := bufio.NewReader(server.Stdout)
	if err != nil {
		//Die?
	}
	//FIXME: type mismatch for line
	for server.running {
		line, pre, err := src.ReadLine()
		if err != nil {
			//Log and continue? Die?
		}

		for pre && err != nil {
			l := ""
			l, pre, err = src.ReadLine()
			line = append(line, l)
		}
		//More error checking here?

		lineStr := string(line)
		//Dispatch the read line to..
		fmt.Print(lineStr) //The server console
		
		if chatRegex.Match(line) { //Irc, if it looks like chat
			sendChat(lineStr)
		} 

		out <- lineStr //The server output queue
	}
	
}

func readConsoleInput() {
	src, err := bufio.NewReader(os.Stdin)
	if err != nil {
		//Die?
	}

	for server.running {
		line, pre, err := src.ReadLine()
		if err != nil {
			//Log and continue? Die?
		}

		for pre && err != nil {
			l := ""
			l, pre, err = src.ReadLine()
			line = append(line, l)
		}
		//More error checking here?
		
		serverIn <- string(line)
	}	
}

func writeServerInput(server Server, in <-chan string) {
	for line := range in {
		fmt.Fprintln(server.Stdin, strings.TrimSpace(line))
	}
}


/*
 [I]2011-08-29 10:45:05 [INFO] * cbeck foo
 [I]2011-08-29 10:48:14 [INFO] <cbeck> lololololol
 */

//var chatRegex regex.Regexp = `\[INFO\] (\* [a-zA-Z0-9\-]+|<[a-zA-Z0-9\-]> ) (.*)`
//var sanitizeRegex regex.Regexp = `\n\r`
func echoIRCToServer(_ string, m ircbot.Message) string {
	sanitized := sanitize.Regex.ReplaceAllString(m.Trailing, " ")

	if m.Ctcp == "" { //Line was normal chat
		serverIn <- fmt.Sprintf("say <%s> %s", m.GetSender(), m.Trailing)
	} else if m.Ctcp == "ACTION"{ //Line was a Ctcp req
		serverIn <- fmt.Sprintf("say * %s %s", m.GetSender(), m.Trailing)
	} //Else ignore

	return ""
}

func (server *Server) echoServerToIRC(chat string) {
	server.bot.Send(&ircbot.Message{
	Command : "PRIVMSG",
	Args : []string{server.IrcChan},
	Trailing : chat
	})
}