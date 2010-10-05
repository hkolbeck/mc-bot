package mcserver

import (
	"sync"
	"os"
	"exec"
	"time"
	"syscall"
	"fmt"
	"io"
)

type Server struct {
	*exec.Cmd
	mutex *sync.Mutex
	running bool
	dir string
}

const STOP_TIMEOUT = 5e9

func StartServer(dir string) (*Server, os.Error) {
	proc, err := exec.Run("/usr/bin/java", []string{"-Xms1024M", "-Xmx1024M", "-jar", "mcs.jar"},
		nil, dir, exec.Pipe, exec.Pipe, exec.MergeWithStdout)

	if err != nil {
		return nil, err
	}

	return &Server{proc, &sync.Mutex{}, true, dir}, nil
}

func (s *Server) Stop(delay int64, msg string) {
	if s == nil || !s.running {
		return
	}

	s.mutex.Lock()
	s.running = false
	if delay < 0 {
		s.Stdin.WriteString("say " + msg)
		time.Sleep(60e9)
	}

	s.Stdin.WriteString("stop\n")
	
	done, alarm := make(chan int), make(chan int)

	go func() {
		s.Wait(0)
		_ = done <- 1
	}()

	go timeout(STOP_TIMEOUT, alarm)

	select {
	case <-done:
	case <-alarm:
		syscall.Kill(s.Pid, 9)
	}
	s.running = false
	s.mutex.Unlock()
}

func (s *Server) BackupState(filename string) (err os.Error) {
	if s == nil || !s.running {
		return os.NewError("Unable to perform backup, server not running")
	}

	s.mutex.Lock()
	s.Stdin.WriteString("say Backup in progress...\n")
	s.Stdin.WriteString("save-off\n")

	time.Sleep(5e9) //Let save-off go through

	bu, err := exec.Run("/u/cbeck/irc-go/mcbot/backup.sh", 
		[]string{" ", filename},
		nil, s.dir, exec.DevNull, exec.Pipe, exec.MergeWithStdout)

	if bu != nil {
		go io.Copy(os.Stdout, bu.Stdout)
		wm, _ := bu.Wait(os.WSTOPPED)
		if ex := wm.ExitStatus(); ex != 0 {
			err = os.NewError(fmt.Sprintf("Backup command returned errorcode: %d", ex))
		}
	}

	s.Stdin.WriteString("save-on\n")
	s.mutex.Unlock()
	
	return
}

func (s *Server) IsRunning() bool {
	return s != nil && s.running
}

func timeout(ns int64, alarm chan int) {
	time.Sleep(ns)
	_ = alarm <- 1
}