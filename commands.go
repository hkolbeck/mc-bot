package main

import (
	"strings"
	"strconv"
	irc "cbeck/ircbot"
	"fmt"
	"io/ioutil"
	"bufio"
	"regexp"
	"exec"
	"net"
	"time"
	"os"
	)

type commandFunc func([]string) []string
type command struct {
	raw string
	sender string
	source int
}

const (
	DefaultStopDelay = 5e9
	notImplemented = "This command is not yet implemented"
)


var commandMap map[string]commandFunc = 
	map[string]commandFunc {
	"?" : helpCmd,
	"backup" : backupCmd,
	"ban" : banCmd,
	"pardon" : pardonCmd,
	"give" : giveCmd,
	"help" : helpCmd,
	"kick" : kickCmd,
	"list" : listCmd,
	"mapgen" : mapgenCmd,
	"restart" : restartCmd,
	"source" : sourceCmd,
	"start" : startCmd,
	"state" : stateCmd,
	"status" : stateCmd,
	"stop" : stopCmd,
	"tp" : tpCmd,
	"version" : versionCmd,
	"whitelist" : whitelistCmd,
}

var commandHelpMap map[string]string = 
	map[string]string {
	"?" : "? [command]: If [command] is present, get usage information on that command, otherwise" +
		" display a list of available commands",

	"backup" : "backup [name]: Force the creation of a persistant backup.  If [name] is present," + 
		" the file will be named 'name.backup', otherwise it will be '<RFC3339 time>.backup'.",

	"ban" : "ban <name or ip> [duration]: Ban a player by ip or name.  If [duration] is present, " +
		"the ban will be lifted after than many minutes. " +
		"If no arguments are passed, get a list of currently banned players and IPs.",

	"pardon" : "pardon <name or ip>: Remove a player from the banned list by name or IP.",

	"give" : "give <player> <item id or name> [num]: Spawn <item> at <player>'s location.  If [num] " +
		"is present, spawn that many of <item>.  Some items may not be spawnable by name.",

	"help" :  "help [command]: If [command] is present, get usage information on that command, otherwise" +
		" display a list of available commands",

	"kick" : "kick <player> [duration]: Kick <player> off the server.  Player will be able to rejoin" + 
		" immediatly unless [duration] is present, in which case they will be banned for that many minutes.",

	"list" : "list: List all players currently connected to the server.",

	"mapgen" : "mapgen [stop]: Force a run of the map generator.  If a mapgen is currently running, get an" +
		" estimate of its progress.",

	"restart" : fmt.Sprintf("restart [delay] [message]: Restart the server after issuing [message] and " +
		"waiting [delay] seconds.  If [delay] is not present, wait %d seconds.", 
		DefaultStopDelay / int64(1e9)),

	"source" : "source: Get information on this bot's source code.",

	"start" : "start: Start the Minecraft server if it's stopped.",

	"state" : "state: Get information on the current server process.",

	"stop" : fmt.Sprintf("stop [delay] [message]: Stop the server after issuing [message] and waiting " +
		"[delay] seconds.  If [delay] is not present, wait %d seconds.", DefaultStopDelay / int64(1e9)),

	"tp" : "tp <player> <destination player>: Teleport <player> to <destination player>'s location.",

	"version" : "version: Get the version number of the currently running minecraft server.",

	"whitelist" : "whitelist <add <name>|remove <name>|list>: Manipulate or examine the server's whitelist.",
}


func directedIRC(cmd string, m *irc.Message) string {
	commands <- &command{cmd, m.GetSender(), SOURCE_IRC}
	return ""
}

func commandDispatch() {
	var reply []string

	for cmd := range commands {
		split := strings.Split(cmd.raw, " ")
		if len(split) < 1 {
			continue
		}

		f, exists := commandMap[split[0]]

		if !exists {
			reply = []string{"Unknown command: " + split[0]}
		} else if !allowed(cmd.sender, split[0], cmd.source) {
			reply = []string{cmd.sender + " is not allowed to invoke '" + split[0] + 
					"'. This incident will be reported."}
		
			logInfo.Printf("%s attempted '%s'\n", cmd.sender, cmd.raw)
		} else {
			//Flush the server output queue first
		Flush: for {
				select {
				case <-commandResponse: 
				default: break Flush
				}
			}
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

func allowed(sender, op string, source int) bool {
	//Is op allowed by default?
	if exists, allowed := config.defaultAccess[op]; exists && allowed {
		return true
	}

	switch source {
	case SOURCE_MC:
		sender = "mc:" + sender
	case SOURCE_IRC:
	
		//TODO: Make sure irc nick is registered

		sender = "irc:" + sender
	}
	
	//If user is marked as part of any groups
	if levels, ok := config.accessLevelMembers[sender]; ok {
		for _, l := range levels {
			level := config.accessLevels[l]
			if exists, allowed := level[op]; exists && allowed {
				return true
			}
		} 
	}

	return false
}



func helpCmd(args []string) []string {
	var reply string
	var ok bool

	if len(args) == 0 {
		for k := range commandHelpMap {
			reply += ", " + k
		}
		reply = "Available commands: " + reply[2:]
	} else if len(args) == 1 {
		reply, ok = commandHelpMap[args[0]]
		if !ok {
			reply = "Unknown command: " + args[0]
		}
	} else {
		reply = "Usage: " + commandHelpMap["help"]
	}

	return []string{reply}
}

func backupCmd(args []string) []string {
	return []string{notImplemented} 
}

func banCmd(args []string) []string {
	if len(args) == 0 || len(args) > 2 {
		return []string{"Usage: " + commandHelpMap["ban"]}
	}

	if !server.IsRunning() {
		return []string{"Server not currently running."}
	}

	var ext string
	isTemp := "."
	//If the thing being banned is an ip, we'll need to append '-ip' to our commands
	if net.ParseIP(args[0]) != nil {
		ext = "-ip"
	}

	if len(args) == 2 {
		dur, err := strconv.Atoi64(args[1])
		if err != nil || dur <= 0{
			return []string{"Could not parse " + args[1] + " as a positive integer."}
		}
		isTemp = fmt.Sprintf(" for %d minute(s).", dur)

		go func() {
			//dur will be in seconds, After() expects nanoseconds
			<-(time.After(dur * 60e9)) 
			server.In <- "pardon" + ext + " " + args[0]
		}()
	}

	server.In <- "ban" + ext + " " + args[0] 
	
	return []string{args[0] + " has been banned" + isTemp}
}

func pardonCmd(args []string) []string {
	if len(args) != 1 {
		return []string{"Usage: " + commandHelpMap["pardon"]}
	}

	if !server.IsRunning() {
		return []string{"Server not currently running."}
	}

	if net.ParseIP(args[0]) != nil {
		server.In <- "pardon-ip " + args[0]
	} else {
		server.In <- "pardon " + args[0]
	}

	return []string{args[0] + " has been pardoned."}
} 

func giveCmd(args []string) []string {
	return []string{notImplemented} 
}

var kickSuccessRegex *regexp.Regexp = regexp.MustCompile(`\[INFO\] CONSOLE: Kicking ([a-zA-Z0-9\-]+)`)
var kickFailureRegex *regexp.Regexp = regexp.MustCompile(`\[INFO\] Can't find user ([a-zA-Z0-9\-]+). No kick.`)
func kickCmd(args []string) []string {
	if len(args) < 1 || 2 < len(args){
		return []string{"Usage: " + commandHelpMap["kick"]}
	}

	if !server.IsRunning() {
		return []string{"Server not currently running."}
	}

	var reply string
	var dur int64 = -1
	var err os.Error

	if len(args) == 2 {
		if dur, err = strconv.Atoi64(args[1]); err != nil || dur <= 0{
			return []string{"Couldn't parse " + args[1] + " as a positive integer."}
		}
	}

	server.In <- "kick " + args[0]

	for line := range commandResponse {
		if match := kickSuccessRegex.FindStringSubmatch(line); match != nil {
			if match[1] == args[0] {
				reply = args[0] + " was kicked"
				break
			}
		} else if match = kickFailureRegex.FindStringSubmatch(line); match != nil {
			if match[1] == args[0] {
				return []string{"Kicking " + args[0] + " failed."}
			}
		}
	}

	if dur != -1 {
		reply = fmt.Sprintf("%s was kickbanned and will be pardoned in %d minute(s).", args[0], dur)
		go func() {
			<-(time.After(dur * 60e9))
			server.In <- "pardon " + args[0]
		}()
	}

	return []string{reply}
}


var listRegex *regexp.Regexp = regexp.MustCompile(`\[INFO\] (Connected players: .*)`)
func listCmd(args []string) []string {
	if !server.IsRunning() {
		return []string{"Server not currently running."}
	}

	server.In <- "list"

	for line := range commandResponse {
		if match := listRegex.FindStringSubmatch(line); match != nil {
			return match[1:]
		}
	}

	return nil 
}



var (
	mapgenRunning bool = false
	lastMapgenOutput string = ""
	lastMapgenRun *time.Time
	)

func mapgenCmd(args []string) []string {
	if mapgenRunning {
		return []string{"MapGen already running, last output: " + lastMapgenOutput}
	}

	if server.IsRunning() {
		server.In <- "save-all"
		server.In <- "save-off"
		for line := range commandResponse {
			if strings.Contains(line, "CONSOLE: Disabling level saving..") {
				break
			}
		}		
	}

	copyWorld(config.MCWorldDir, config.MapTempWorldDir)
	server.In <- "save-on"
	mapgenRunning = true
	lastMapgenRun = time.LocalTime()
	
	command := exec.Command(config.MapUpdateCommand.Command, config.MapUpdateCommand.Args...)
	
	//These two lambdas will constantly be racing for lastMapgenOutput, and that's ok
	go func() {
		out, _ := command.StdoutPipe()	
		outBuf := bufio.NewReader(out)
		for {
			line, _, err := outBuf.ReadLine()
			if err != nil {
				return
			} else if len(line) < 1 {
				continue
			}
			fmt.Printf("%s\n", line)
			lastMapgenOutput = string(line)
		}
	}()
	go func() {
		err, _ := command.StderrPipe()
		errBuf := bufio.NewReader(err)

		for {
			line, _, err := errBuf.ReadLine()
			if err != nil {
				return
			} else if len(line) < 1 {
				continue
			}
			fmt.Printf("%s\n", line)
			lastMapgenOutput = string(line)
		}
	}()

	go func() {
		err := command.Run()

		if err != nil {
			bot.Send(&irc.Message{
			Command: "PRIVMSG",
			Args: []string{config.IrcChan},
			Trailing: "MapGen exited uncleanly: " + err.String(),
			})
		} else {

			dur := time.Seconds() - lastMapgenRun.Seconds()

			bot.Send(&irc.Message{
			Command: "PRIVMSG",
			Args: []string{config.IrcChan},
			Trailing: fmt.Sprintf("MapGen Complete in %02d:%02d:%02d", dur / 3600, (dur % 3600) / 60, dur % 60),
			})		
		}

		mapgenRunning = false
		lastMapgenOutput = ""
	}()

	return []string{"MapGen started"} 
}

func restartCmd(args []string) []string {
	stopCmd(args)
	return startCmd(nil)
}

func sourceCmd(args []string) []string {
	return []string{"MCBot was written by Cory 'cbeck' Kolbeck.  Its source and license" +
			" information can be found at https://github.com/ckolbeck/mc-bot"}
}

var versionRegex *regexp.Regexp = regexp.MustCompile(`\[INFO\] Starting (minecraft server version .*)`)
func startCmd(args []string) []string {
	if len(args) != 0{
		return []string{"Usage: " + commandHelpMap["start"]}
	}

	if err := server.Start(); err != nil {
		return []string{err.String()}
	}

	for line := range commandResponse {
		if match := versionRegex.FindStringSubmatch(line); match != nil {
			serverVersion = match[1]
			break
		}
	}

	return []string{"Server started."}
}

func stateCmd(args []string) (reply []string) {
	var lines []string
	if len(args) != 0{
		return []string{"Usage: " + commandHelpMap["state"]}
	}

	//GetPID will return an error if server is not running
	pid, err := server.GetPID()
	if err != nil {
		return []string{err.String()}
	}

	switch config.HostOS {
	case "linux":
		raw, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
		if err != nil {
			reply = []string{"Error while assessing status: " + err.String()}
		}		
		lines = strings.Split(string(raw), "\n")
	case "windows":
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("pid eq %d", pid), "/FO", "LIST")
		raw, err := cmd.Output()
		if err != nil {
			reply = []string{"Error while assessing status: " + err.String()}
		}
		lines = strings.Split(string(raw), "\r\n")
	}

	stats := make(map[string]string, 24)
	for _, line := range lines {
		split := strings.Split(line, ":")
		stats[split[0]] = line
	}

	switch config.HostOS {
	case "linux":
		reply = append(reply, stats["VmSize"])
		reply = append(reply, stats["VmSwap"])
		reply = append(reply, stats["Threads"])
	case "windows":
		reply = append(reply, stats["Mem Usage"])
		reply = append(reply, stats["Status"])
	}

	reply = append(reply, fmt.Sprintf("Errors: %d", serverErrors))
	reply = append(reply, fmt.Sprintf("Severe Errors: %d", severeServerErrors))
	reply = append(reply, serverVersion)
	
	if mapgenRunning {
		reply = append(reply, "MapGen currently running: " + lastMapgenOutput)
	} else if lastMapgenRun != nil {
		reply = append(reply, "MapGen last run  " + lastMapgenRun.Format("Mon Jan _2 15:04"))
	} else {
		reply = append(reply, "No MapGen run since last bot restart.")
	}

	return 
}

func stopCmd(args []string) []string {
	var delay int64
	var msg string

	if !server.IsRunning() {
		return []string{"Server not currently running."}
	}

	if len(args) == 0 {
		delay = DefaultStopDelay
		msg = fmt.Sprintf("Stop command issued, going down in %d seconds.", delay / 1e9) 
	} else {
		if d, err := strconv.Atoi64(args[0]); err == nil {
			delay = d * 1e9
			args = args[1:]
		} else {
			delay = DefaultStopDelay
		}

		if len(args) > 0 {
			msg = strings.Join(args, " ")
		} else { 
			msg = fmt.Sprintf("Stop command issued, going down in %d seconds.", delay / 1e9) 
		}
	} 

	serverErrors = 0
	severeServerErrors = 0
	serverVersion = ""

	if err := server.Stop(delay, msg); err != nil {
		return []string{err.String()}
	}

	return []string{"Server stopped."}
}
/*
 tp cbeck foo
 2011-11-26 01:27:45 [INFO] Can't find user foo. No tp.
 tp foo cbeck
 2011-11-26 01:27:50 [INFO] Can't find user foo. No tp.
 tp cbeck akhiros
 2011-11-26 01:28:30 [INFO] CONSOLE: Teleporting cbeck to akhiros.
 [INFO] User cbeck and akhiros are in different dimensions. No tp.
 */
var tpRegex *regexp.Regexp = regexp.MustCompile(`\[INFO\] (CONSOLE:)? (Teleporting.*|Can't find user.*)`)
func tpCmd(args []string) []string {
	if len(args) != 2 {
		return []string{"Usage: " + commandHelpMap["tp"]}
	}

	if !server.IsRunning() {
		return []string{"Server not currently running."}
	}

	server.In <- fmt.Sprintf("tp %s %s", args[0], args[1])

	for line := range commandResponse {
		if match := tpRegex.FindStringSubmatch(line); match != nil {
			return []string{match[2]}
		}
	}

	//Shouldn't be reachable
	return nil
}

func versionCmd(args []string) []string {
	if serverVersion != "" {
		return []string{serverVersion}
	}
	return []string{"Server not running or version unknown."}
}

/*
whitelist remove cbeck
 2011-11-26 01:51:38 [INFO] CONSOLE: Removed cbeck from white-list
whitelist add cbeck
 2011-11-26 01:52:03 [INFO] CONSOLE: Added cbeck to white-list
 --
[INFO] White-listed players:
*/
var whitelistRegex *regexp.Regexp = regexp.MustCompile(`\[INFO\] CONSOLE: (Removed .+ from white-list|Added .+ to white-list)`)
func whitelistCmd(args []string) (reply []string) {
	if len(args) == 0 {
		return []string{"Usage: " + commandHelpMap["whitelist"]}
	}

	if !server.IsRunning() {
		return []string{"Server not currently running."}
	}
	
	switch args[0] {
	case "add", "remove":
		if len(args) < 2 {
			return []string{args[0] + " requires at least one argument"}
		}

		for _, name := range args[1:] {
			server.In <- fmt.Sprintf("whitelist %s %s", args[0], name)
			for line := range commandResponse {
				if match := whitelistRegex.FindStringSubmatch(line); match != nil {
					reply = append(reply, match[1])
					break
				}
				 
			}
		}
	case "list":
		server.In <- "whitelist list"
		for line := range commandResponse {
			if index := strings.Index(line, "White-listed players:"); index != -1 {
				reply = append(reply, line[index:])
				break
			}	
		}
	default:
		return []string{"Usage: " + commandHelpMap["whitelist"]}
	}

	return
}

