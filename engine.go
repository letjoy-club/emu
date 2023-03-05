package main

import (
	"fmt"
	"sync"
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

	service.runner.Stop()
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
