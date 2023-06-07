package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/samber/lo"
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

	fdNum       int
	mem         int
	cpu         float64
	connections []string
	paths       []string
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
		{Name: fmt.Sprintf("%s-%s.%s.log", r.exec, r.mode, Stdout)},
		{Name: fmt.Sprintf("%s-%s.%s.log", r.exec, r.mode, Stderr)},
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
		time.Sleep(time.Millisecond * 100)
		defer r.onStop()
		if err := r.cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill", err)
		}
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

	handlers, _ := r.process.OpenFiles()
	paths := []string{}
	for _, handler := range handlers {
		paths = append(paths, handler.Path)
	}
	r.paths = paths

	fdNum, _ := r.process.NumFDs()
	r.fdNum = int(fdNum)

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

var re = regexp.MustCompile(`@(\S+)`)

func processConfig(args []string, meta map[string]string) {
	if args == nil {
		return
	}
	lo.ForEach(args, func(arg string, i int) {
		results := re.FindAllString(arg, -1)
		if len(results) >= 1 {
			for _, result := range results {
				err := ReplaceMetaFile(result, meta)
				if err != nil {
					continue
				}
			}
		}
	})
}

func ReplaceMetaFile(file string, metaVars map[string]string) error {
	filePath := strings.TrimPrefix(file, "@")
	data, err := os.ReadFile(path.Join("service", filePath))
	if err != nil {
		return err
	}
	keys := lo.Keys(metaVars)
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})

	for _, key := range keys {
		value := metaVars[key]
		data = bytes.ReplaceAll(data, []byte(key), []byte(value))
	}
	os.WriteFile(path.Join("service", "@"+filePath), data, 0644)
	return nil
}

func NewRunner(service *Service, mode Mode, meta map[string]string) *Runner {
	processConfig(service.Args, meta)
	os.Chmod(service.ExecPath(), 0777)
	exe := service.Exec
	if !strings.HasPrefix(service.Exec, "./") {
		exe = "./" + service.Exec
	}
	cmd := exec.Command(exe, service.Args...)
	fmt.Println("$", exe, strings.Join(service.Args, " "))
	for _, env := range service.Env {
		fmt.Println(" > ", env)
	}
	if service.Packed() {
		cmd.Dir = service.ServiceFolder()
	} else {
		cmd.Dir = "service"
	}
	envs := append(os.Environ(), service.Env...)
	cmd.Env = envs
	cmd.SysProcAttr = ProcAttr()

	return &Runner{
		cmd:  cmd,
		name: service.Name,
		exec: service.Exec,
		mode: mode,

		onStart: func() { service.Running = true },
		onStop:  func() { service.Running = false },

		connections: []string{},
		paths:       []string{},
	}
}
