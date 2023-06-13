//go:build windows
// +build windows

package platform

import "github.com/gdamore/tcell/v2"

// Suspend is disabled in Windows.
func Suspend(t tcell.Screen) {
}
