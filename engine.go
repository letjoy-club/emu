package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Engine struct {
	services []*Service

	lock sync.Mutex
	mode Mode
}

var ErrServiceNotFound = fmt.Errorf("service not found")

func (e *Engine) Init(mode Mode, services []*Service) {
	for _, s := range services {
		runner := NewRunner(s, mode)
		runner.Start()
		s.runner = runner
	}
	e.services = services

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		e.lock.Lock()
		wg := sync.WaitGroup{}
		wg.Add(len(e.services))
		for _, s := range e.services {
			go func(s *Service) {
				s.runner.Stop()
				wg.Done()
			}(s)
		}
		wg.Wait()

		e.lock.Unlock()
		os.Exit(0)
	}()
}

func (e *Engine) GetService(exec string) *Service {
	for _, s := range e.services {
		if s.Exec == exec {
			return s
		}
	}
	return nil
}

func (e *Engine) StartService(exec string) error {
	service := e.GetService(exec)
	if service == nil {
		return ErrServiceNotFound
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	if err := service.runner.Stop(); err != nil {
		fmt.Println("failed to stop service", exec, err)
	}

	service.runner = NewRunner(service, e.mode)
	return service.runner.Start()
}

func (e *Engine) StopService(exec string) error {
	service := e.GetService(exec)
	if service == nil {
		return ErrServiceNotFound
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	return service.runner.Stop()
}

func (e *Engine) Restart(exec string) error {
	service := e.GetService(exec)
	if service == nil {
		return ErrServiceNotFound
	}
	service.runner.Stop()
	service.runner = NewRunner(service, e.mode)
	return service.runner.Start()
}
