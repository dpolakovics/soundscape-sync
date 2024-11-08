//go:build windows

package logic

import (
  "syscall"
)

func getSysProcAttr() *syscall.SysProcAttr {
  return &syscall.SysProcAttr{Hidewindow: true}
}
