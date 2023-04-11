package main

import (
	"syscall"
)

func ProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: false,
	}
}
