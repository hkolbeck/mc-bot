//Copyright 2010 Cory Kolbeck <ckolbeck@gmail.com>.
//So long as this notice remains in place, you are welcome 
//to do whatever you like to or with this code.  This code is 
//provided 'As-Is' with no warrenty expressed or implied. 
//If you like it, and we happen to meet, buy me a beer sometime

package main

import (
	"cbeck/ircbot"
	"cbeck/mcserver"
	"time"
	"strings"
	"os"
	"fmt"
	"log"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"bufio"
	"container/vector"
)

var (
	bot *ircbot.Bot
	server *mcserver.Server
	logerr *log.Logger = log.New(os.Stderr, "[E]", log.Ldate | log.Ltime)
	var loginfo *log.Logger = log.New(os.Stdout, "[I]", 0)
)


func session() {
	defer ircbot.RecoverWithTrace()
	bot = ircbot.NewBot("MC-Bot", '!')

	bot.SetPrivmsgHandler(parseCommand, echoChat)
	_, e := bot.Connect(network, 6667, []string{"#minecraft"})

	if e != nil {
		panic(e.String())
	}

	e = parseItems(itemFile)

	if e != nil {
		logerr.Print("[E] Error loading items: " + e.String())
		os.Exit(1)
	}
	
	server, e = mcserver.StartServer(mcDir, loginfo, logerr) 
	
	if e != nil {
		logerr.Print("[E] Error creating server")
		panic(e.String())
	}

	go autoBackup(server)
	go monitorOutput(server)
	go io.Copy(server.Stdin, os.Stdin)
	
	defer func(s *mcserver.Server) {
		s.Stop(1e9,"Crash Intercepted, server going down NOW")
	}(server)

	select {}
}

var sanitizeRegex *regexp.Regexp = regexp.MustCompile(`[^ -~]`)

func parseCommand(c string, m *ircbot.Message) string {
	sender := m.GetSender()
	
	if (ignored[sender] && sender != admin) || (!freeForAll && !trusted[sender]) {
		return ""
	}

	c = sanitizeRegex.ReplaceAllString(c, "_")
		
	var args []string
	split := strings.Split(strings.TrimSpace(c), " ", 2)
	command := strings.ToLower(split[0])
	if len(split) > 1 {
		args = strings.Split(split[1], " ", -1)
	}

	switch command {
	case "give":
		return give(c)
	
	case "parseitems" :
		if e := parseItems(itemFile); e != nil {
			return e.String()
		}
		return "Done, " + sender
	
	case "restart":
		return restart(sender)
	
	case "backup":
		return backup(args)
	
	case "state":
		return state()
	
	case "stop":
		return stop(sender)
	
	case "halt" :
		if !trusted[sender] {
			return ""
		}	
		server.Stop(1e9,"Server going down NOW!")
		return "Server halted"
	
	case "tp" :
		return tp(args)
	
	case "ignore" : 
		return ignore(args, sender)
	
	case "trust" :
		return trust(args, sender)
	
	case "list" :
		listReq = true
		server.Stdin.WriteString("\nlist\n")
		return <-lastList
	
	case "ffa" :
		if !trusted[sender] {
			return ""
		}
		freeForAll = !freeForAll

	case "additem" :
		return addItem(args, sender)

	case "help" :
		return "give | restart | list | backup | state | stop | tp | source | help"

	case "mc-bot", "source" : 
		return "MC-Bot was written by Cory Kolbeck. Its source can be found at http://github.com/ckolbeck/mc-bot"
	}
	return "Huh?"
}

func addItem(args []string, sender string) string {
	if !trusted[sender] {
		return ""
	}

	if len(args) < 2 {
		return "Expected format `additem <name> <itemid>`"
	}

	id, err := strconv.Atoi(args[len(args) - 1])

	if err != nil {
		return "Couldn't parse `" + args[len(args) - 1] + "` as an int"
	}

	itemName := strings.ToLower(strings.Join(args[:len(args) - 1], " "))

	if _, exists := items[itemName]; exists && sender != admin {
		return "Item already exists."
	}

	items[itemName] = id

	return "Okay, " + sender
}


func echoChat(c string, m *ircbot.Message) string {
	if server == nil || !server.IsRunning() {
		return ""
	}

	c = sanitizeRegex.ReplaceAllString(c, "_")

	fmt.Fprintf(server.Stdin, "say <%s> %s\n", m.GetSender(), c)

	return ""
}

func stop(sender string) string {
	if !trusted[sender] {
		return ""
	}

	if !server.IsRunning() {
		return "The server is not currently running"
	}
	server.Stop(10e9, fmt.Sprintf("Server halt requested by %s, going down in 10s\n", sender))
	server = nil
	return "Server halted."
}

var giveRegex *regexp.Regexp = regexp.MustCompile(`give[ \t]+([a-zA-Z0-9_]+)[ \t]+([a-zA-Z\- ]+|[0-9]+)[ \t]*([0-9]+)?`)

func give(cmd string) string {
	var num int = 1
	var id int
	var err os.Error
	var ok bool

	if !server.IsRunning() {
		return "The server is not currently running"
	}

	if match := giveRegex.FindStringSubmatch(cmd); match != nil {
		match[2] = strings.TrimSpace(strings.ToLower(match[2]))

		id, err = strconv.Atoi(match[2])
		
		if err != nil {
			id, ok = items[match[2]]
			
			if !ok {
				return fmt.Sprintf("Unknown item: '%s'.  Did you mean: %v", 
					match[2], strings.Join(notFound(match[2]), ", "))
			}
		}

		if len(match) == 4 && match[3] != "" {
			num, err = strconv.Atoi(match[3])
			if err != nil {
				return "Couldn't parse '" + match[3] + "' as a quantity."
			}
		}

		for i := num; i > 0; i -= 64 {
			fmt.Fprintf(server.Stdin, "give %s %d %d\n", match[1], id, min(64, i))
		}

		return ""
	}

	return "Expected format: `give <playername> <objectid | objectname> [num]`"
}

func notFound(query string) []string {
	ans := make(vector.StringVector, 0, 20)
	
	for _, w := range strings.Split(query, " ", -1) {
		if w == "" || w == " " {
			continue
		}

		for key := range items {
			if p := strings.Index(key, w); p != -1 {
				ans.Push(key)
			}
		}
	}

	return ans
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func restart(sender string) string {
	if !trusted[sender] {
		return ""
	}

	var err os.Error

	server.Stop(10e9, fmt.Sprintf("Server restart requested by %s, going down in 10s\n", sender))
	server, err = mcserver.StartServer(mcDir, loginfo, logerr )

	if err != nil {
		return "Could not start server: " + err.String()
	}

	go autoBackup(server)
	go monitorOutput(server)
	go io.Copy(server.Stdin, os.Stdin)

	return "Server restarted"
}

func backup(args []string) string {
	if !server.IsRunning() {
		return "The server is not currently running"
	}
	
	var bkfile string

	if len(args) > 0 {
		bkfile = args[0] + ".tgz"
	} else {
		bkfile = time.LocalTime().Format("2006-01-02T15_04_05") + ".tgz"
	}

	err := server.BackupState(bkfile)

	if err != nil {
		return "Error attempting to perform backup: " + err.String()
	}

	return "Backup finished"
}

func state() string {
	if !server.IsRunning() {
		return fmt.Sprintf("The server is not currently running")
	}

	usage, err := getMemUsage()

	if err != nil {
		return err.String()
	}

	return fmt.Sprintf("Server is up and currently using %dK virtual memory", usage)
}

var memRegex *regexp.Regexp = regexp.MustCompile("VmSize:[^0123456789]*([0123456789]+)")

func getMemUsage() (int, os.Error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", server.Pid), os.O_RDONLY, 0444)
	
	if err != nil {
		return -1, os.NewError("Error opening status file: " + err.String())
	}

	defer f.Close()

	raw, err := ioutil.ReadAll(f)
	
	if err != nil {
		return -1, os.NewError("Error reading status file")
	}
	
	mtch := memRegex.FindSubmatch(raw)

	if mtch == nil {
		return -1, os.NewError("Error in regexp parsing of status file")
	}

	usage, err := strconv.Atoi(string(mtch[1]))

	if err != nil {
		return -1, os.NewError("Error parsing status file")
	}

	return usage, nil
}

func tp(args []string) string {
	if len(args) == 2 {
		fmt.Fprintf(server.Stdin, "tp %s %s\n", args[0], args[1])
	} else {
		return "Expected format: `tp <player> <target-player>`"
	} 

	return ""
}

func ignore(args []string, sender string) string {
	if !trusted[sender] {
		return ""
	}
	
	ign := "Ignoring: "
	unign := "Unignoring: "
	
	for _, i := range args {
		if (!trusted[i] || sender == admin) && i != admin {
			if ignored[i] {
				unign += i + " "
				ignored[i] = false, false
			} else {
				ign += i + " "
				ignored[i] = true
			}
		} 
	}

	return ign + unign
}

func trust(args []string, sender string) string {
	if sender != admin {
		return ""
	}
	
	trst := "Trusting: "
	untrst := "Untrusting: "
	
	for _, i := range args {
		if i != admin {
			if trusted[i] {
				untrst += i + " "
				trusted[i] = false, false
			} else {
				trst += i + " "
				trusted[i] = true
			}
		}
	}

	return trst + untrst
}

func autoBackup(s *mcserver.Server) {
	tick := time.Tick(3610e9)
	for s.IsRunning() {
		t := time.LocalTime()
		s.BackupState(fmt.Sprintf("%d.tgz", t.Hour))
		<-tick
	}
}

var listRegex *regexp.Regexp = regexp.MustCompile(`Connected players:.*`)
var msgRegex *regexp.Regexp = regexp.MustCompile(`\[INFO\] <[a-zA-Z0-9_]+>`)

func monitorOutput(s *mcserver.Server) {
	defer ircbot.RecoverWithTrace()

	in := bufio.NewReader(s.Stdout)

	for str, err := in.ReadString('\n'); s.IsRunning() && err == nil; str, err = in.ReadString('\n') {
		if listReq && listRegex.MatchString(str) {
			lastList <- str[27:]
			listReq = false
		} else if msgRegex.MatchString(str) {
			bot.Send(network, &ircbot.Message{
			Command : "PRIVMSG",
			Args : []string{"#minecraft"},
			Trailing : str[27:],
			})
		}

		loginfo.Print(str)
	}
}
