//Copyright 2010 Cory Kolbeck <ckolbeck@gmail.com>.
//So long as this notice remains in place, you are welcome 
//to do whatever you like to or with this code.  This code is 
//provided 'As-Is' with no warrenty expressed or implied. 
//If you like it, and we happen to meet, buy me a beer sometime

package main

import (
	"cbeck/ircbot"
	"cbeck/mcserver"
	"os"
	"log"
)

var (
	bot *ircbot.Bot
	server *mcserver.Server
	config *Config
	logErr *log.Logger = log.New(os.Stderr, "[E]", log.Ldate | log.Ltime)
	logInfo *log.Logger = log.New(os.Stdout, "[I]", 0)
)


func main() {
}

