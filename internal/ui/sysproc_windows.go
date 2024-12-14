//go:build windows
// +build windows

package ui

import (
    "syscall"
)

const CREATE_NO_WINDOW = 0x08000000

func getSysProcAttr() *syscall.SysProcAttr {
    return &syscall.SysProcAttr{
        CreationFlags: CREATE_NO_WINDOW,
    }
}