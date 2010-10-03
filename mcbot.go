package main

import (
	"./ircbot"
	"./mcserver"
)

var bot *ircbot.Bot
var server *mcserver.Server

func main() {
	for {
		session()
		time.Sleep(10e9)
	}
}

func session() {
	defer ircbot.RecoverWithTrace()
	bot = ircbot.NewBot(McBot, '!')

	bot.SetPrivmsgHandler(parseCommand, nil)
	_, e := bot.Connect("irc.cat.pdx.edu", 6667, []string{"#minecraft"})

	if e != nil {
		panic(e.String())
	}
	
	server = mcserver.StartServer() 

	defer func() {
		server.Stop(0)
	}()

	select {}
}

func parseCommand(c string, m *ircbot.Message) string {
	var args []string
	split := strings.Split(strings.TrimSpace(c), " ", 2)
	command := strings.ToLower(split[0])
	if len(split) > 1 {
		args = strings.Split(split[1], " ", -1)
	}

	switch command {
	case "give":
		return give(args)
	case "restart":
		return restart(m)
	case "backup":
		return backup()
	case "state":
		return state()
	case "say":
		return say(args)
	case "tp" :
		return tp(args)
	}

	return "Huh?"
}


func give(args []string) string {
	if !server.IsRunning() {
		return "The server is not currently running"
	}

	if len(args) == 2 { 
		fmt.Fprintf(server.Stdin, "give %s %s %s\n", args[0], args[1], "1")
	} else if len(args) == 3 {
		fmt.Fprintf(server.Stdin, "give %s %s %s\n", args[0], args[1], args[2])
	} else {
		return "Expected format: `give <playername> <objectid> [num]`"
	}
	
	return ""
}

func restart(m *ircbot.Message) string {
	server.Stop(10e9, fmt.Sprintf("Server restart requested by %s, going down in 10s\n", m.GetSender()))
	server, err := mcserver.StartServer("/disk/trump/cbeck")

	if err != nil {
		return "Could not start server: " + err.String()
	}

	return "Server restarted"
}

func backup() string {
	if !server.IsRunning() {
		return "The server is not currently running"
	}

	err := server.BackupState()

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

	return fmt.Sprintf("Server is up currently using %dK virtual memory", usage)
}

func getMemUsage() (int, os.Error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", server.Pid), os.O_RDONLY, 0444)
	
	if err != nil {
		return -1, os.NewError("Error opening status file: " + err.String())
	}

	defer f.Close()
	reader := bufio.NewReader(f)

	for i := 0; i < 11; i++ {
		reader.ReadBytes('\n')
	}

	mstr, _ := reader.ReadString('\n')

	usage := 0

	read, _ := fmt.Scanf("VmSize:  %d", &usage)

	if r < 1 {
		return -1, os.NewError("Error parsing status file on line: " + mstr) 
	}

	return usage, nil
}

func say(args []string) string {
	if args == nil {
		return ""
	}

	Fprint(server.Stdin, strings.Join(args, " "))
	return ""
}

func tp(args []string) string {
	if len(args) == 2 {
		Fprintf(server.In, "tp %s %s\n", args[0], args[1])
	} else {
		return "Expected format: `tp <player> <target-player>`"
	} 

	return ""
}


func startAutoBackup(interval int64, numbackups int) {
	tick := time.Tick(interval)

	go func() {
		for {
			for i := 0; i < numbackups; i++ {
				server.BackupState(fmt.Sprintf("%d.tgz", i))
				<-tick
			}
		}
	}()
}