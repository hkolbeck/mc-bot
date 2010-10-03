package mcserver

import (
	"sync"
	"os"
	"exec"
	"time"
	"syscall"
	)

type Server struct {
	*exec.Cmd
	mutex *sync.Mutex
	running bool
	dir string
	started *time.Time
}

type ServerStats struct {
	uptime 
}

const STOP_TIMEOUT = 5e9

func StartServer(dir string) (*Server, os.Error) {
	proc, err := exec.Run("java", []string{"-Xms1024M", "-Xmx1024M", "-jar mcs.jar"},
		nil, dir, exec.Pipe, exec.Pipe, exec.MergeWithStdout)

	if err != nil {
		return nil, err
	}

	return &Server{proc, &sync.Mutex{}, true, dir, time.GetLocalTime()}, nil
}

func (s *Server) Stop(delay int64, msg string) {
	if s == nil || !s.running {
		return
	}

	s.mutex.Lock()
	s.running = false
	if delay > 0 {
		s.Stdin.WriteString(msg)
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

	bu, err := exec.Run("tar", 
		[]string{"-czf mc_server_backups/" + filename, "banned-ips.txt", "server.log", 
		"server.properties", "banned-players.txt",  "ops.txt",  "server.log.lck",  "world"},
		nil, s.dir, exec.DevNull, exec.DevNull, exec.DevNull)

	if bu != nil {
		wm, err := bu.Wait(os.WSTOPPED)
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