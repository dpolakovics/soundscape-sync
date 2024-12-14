//go:build windows
// +build windows

package ui

import "syscall"

func getSysProcAttr() *syscall.SysProcAttr {
    return &syscall.SysProcAttr{
        CreationFlags: syscall.CREATE_NO_WINDOW,
    }
}