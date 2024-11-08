//go:build windows

import (
  "syscall"
)

func getSysProcAttr() *syscall.SysProcAttr {
  return &syscall.SysProcAttr{Hidewindow: true}
}
