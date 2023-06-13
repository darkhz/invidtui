//go:build !windows
// +build !windows

package platform

import (
	"syscall"

	"github.com/gdamore/tcell/v2"
)

// Suspend suspends the application.
func Suspend(t tcell.Screen) {
	t.Suspend()
	syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	t.Resume()
}
