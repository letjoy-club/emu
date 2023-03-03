package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Runner struct {
	name string
	exec string
	cmd  *exec.Cmd

	mode Mode

	onStart func()
	onStop  func()
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
	log.SetOutput(&lumberjack.Logger{
		Filename:   fmt.Sprintf("log/%s-%s.%s.log", r.exec, r.mode, channel),
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
	})
	for {
		if n, err := reader.Read(content); err != nil {
			wg.Done()
			return err
		} else {
			str := string(content[:n])
			fmt.Println(str)
			log.Println(str)
		}
	}
}

func (r *Runner) Stop() error {
	if r.cmd.Process != nil {
		if r.cmd.ProcessState != nil && r.cmd.ProcessState.Exited() {
			return nil
		}
		r.cmd.Process.Signal(os.Interrupt)
		time.Sleep(time.Second)
		defer r.onStop()
		if r.cmd.ProcessState == nil || r.cmd.ProcessState.Exited() {
			return nil
		}
		return r.cmd.Process.Kill()
	}
	return nil
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
	return r.cmd.Start()
}

func NewRunner(service *Service, mode Mode) *Runner {
	exe := service.Exec
	if !strings.HasPrefix(service.Exec, "./") {
		exe = "./" + service.Exec
	}
	cmd := exec.Command(exe, service.Args...)
	cmd.Dir = "service"
	cmd.Env = service.Env

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
	}
}
