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


func main() {
	defer ircbot.RecoverWithTrace()
	bot = ircbot.NewBot("MC-Bot", '!')

}

