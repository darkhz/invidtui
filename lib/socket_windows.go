//go:build windows
// +build windows

package lib

func getSocket(sock string) string {
	return `\\.\pipe\invidtui-socket`
}
