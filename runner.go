package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Runner struct {
	name string
	exec string
	cmd  *exec.Cmd

	process *process.Process
	mode    Mode

	onStart func()
	onStop  func()

	mem         int
	cpu         float64
	connections []string
	lastCheck   time.Time
}

type Channel string

var (
	Stdout Channel = "stdout"
	Stderr Channel = "stderr"
)

type Mode string

var (
	Staging Mode = "staging"
	Prod    Mode = "prod"
)

type File struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

func (r *Runner) LogFiles() []File {
	files := []File{
		{
			Name: fmt.Sprintf("%s-%s.%s.log", r.exec, r.mode, Stdout),
		},
		{
			Name: fmt.Sprintf("%s-%s.%s.log", r.exec, r.mode, Stderr),
		},
	}
	ret := []File{}
	for _, file := range files {
		fs, err := os.Stat("log/" + file.Name)
		if err != nil {
			continue
		}
		file.Size = int(fs.Size())
		ret = append(ret, file)
	}
	return ret
}

func (r *Runner) read(reader io.ReadCloser, channel Channel, wg *sync.WaitGroup) error {
	defer reader.Close()
	defer fmt.Println(r.exec, "reader closed")
	content := make([]byte, 1024*2)
	loggerOut := &lumberjack.Logger{
		Filename:   fmt.Sprintf("log/%s-%s.%s.log", r.exec, r.mode, channel),
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
	}
	logger := log.New(loggerOut, "", log.LstdFlags)
	for {
		if n, err := reader.Read(content); err != nil {
			wg.Done()
			return err
		} else {
			str := string(content[:n])
			hub.msgC <- Msg{Content: str, Channel: r.exec}
			logger.Println("> ", str)
		}
	}
}

func (r *Runner) Stop() error {
	if r.cmd.Process != nil {
		defer func() { r.cmd.Process = nil }()
		if err := r.cmd.Process.Signal(os.Interrupt); err != nil {
			fmt.Println("failed to send signal", err)
		}
		time.Sleep(time.Millisecond * 500)
		defer r.onStop()
		if err := r.cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill", err)
		}
		// r.cmd.Process.Wait()
		time.Sleep(time.Millisecond * 100)

		if err := syscall.Kill(-r.cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return err
		}
		r.cmd.Process.Release()
	}
	return nil
}

func (r *Runner) checkStat() {
	if r.process == nil {
		return
	}
	if !time.Now().After(r.lastCheck.Add(4 * time.Second)) {
		return
	}

	mem, err := r.process.MemoryInfo()
	if err == nil {
		r.mem = int(mem.RSS)
	}
	cpu, _ := r.process.CPUPercent()
	r.cpu = cpu

	conns, err := r.process.Connections()
	if err == nil {
		addresses := []string{}
		for _, conn := range conns {
			addresses = append(addresses, fmt.Sprintf("%s:%d", conn.Laddr.IP, conn.Laddr.Port))
		}
		r.connections = addresses
	} else {
		r.connections = []string{}
	}

	r.lastCheck = time.Now()
}

func (r *Runner) Start() error {
	if r.cmd.Process != nil && r.cmd.ProcessState != nil && r.cmd.ProcessState.Exited() {
		return nil
	}
	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go r.read(stderr, Stderr, &wg)
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	go r.read(stdout, Stdout, &wg)

	go func() {
		wg.Wait()
		r.onStop()
	}()
	r.onStart()
	if err := r.cmd.Start(); err != nil {
		return err
	}
	if r.cmd.Process != nil {
		p, err := process.NewProcess(int32(r.cmd.Process.Pid))
		if err != nil {
			return err
		}
		r.process = p
	}
	return nil
}

func NewRunner(service *Service, mode Mode) *Runner {
	os.Chmod(service.ExecPath(), 0777)
	exe := service.Exec
	if !strings.HasPrefix(service.Exec, "./") {
		exe = "./" + service.Exec
	}
	cmd := exec.Command(exe, service.Args...)
	if service.Packed() {
		cmd.Dir = filepath.Join("service", service.Folder)
	} else {
		cmd.Dir = "service"
	}
	envs := append(os.Environ(), service.Env...)
	cmd.Env = envs
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	return &Runner{
		cmd:  cmd,
		name: service.Name,
		exec: service.Exec,
		mode: mode,
		onStart: func() {
			service.Running = true
		},
		onStop: func() {
			service.Running = false
		},
		connections: []string{},
	}
}
