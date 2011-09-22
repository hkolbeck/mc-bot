package main

import (
	mc "cbeck/mcserver"
	"bufio"
	"fmt"
	"log"
	"os"
)

var logInfo, logErr *log.Logger

func main() {
	logInfo = log.New(os.Stdout, "[I] ", log.LstdFlags | log.Lshortfile)
	logErr = log.New(os.Stderr, "[E] ", log.LstdFlags | log.Lshortfile)


	server, err := mc.NewServer("java", 
		[]string{"-Xms1024M", "-Xmx1024M", "-jar", "/home/cbeck/mc/minecraft_server.jar", "nogui"},
		logInfo, logErr)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	go readIn(server.In, server)
	go writeOut(server.Out)
	go writeErr(server.Err)

	fmt.Println("Main Sleeping")

	select{}
}

func readIn(pipe chan<- string, server *mc.Server) {
	in := bufio.NewReader(os.Stdin)

	for {
		line, _, err := in.ReadLine()
		if err != nil {
			logErr.Println(err)
		} else if len(line) < 1 {
			continue
		}
		
		if line[0] == '!' {
			switch string(line[1:]) {
			case "start":
				if err := server.Start(); err != nil {
					logErr.Println(err)					
				}
			case "stop":
				if err := server.Stop(0, ""); err != nil {
					logErr.Println(err)
				}
			case "exit":
				if err := server.Destroy(); err != nil {
					logErr.Println(err)
				}
				os.Exit(0)
			default:
				fmt.Fprintf(os.Stderr, "[E] Unrecognized command: %s\n", line)
			}
		} else {
			pipe <- string(line)
		}
	}
}

func writeOut(pipe <-chan string) {
	for l := range pipe {
		fmt.Printf("[SI] %s\n", l)
	}
}

func writeErr(pipe <-chan string) {
	for l := range pipe {
		fmt.Fprintf(os.Stderr, "[SE] %s\n", l)
	}
}