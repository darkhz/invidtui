//go:build windows
// +build windows

package platform

// Socket returns the socket path.
func Socket(sock string) string {
	return `\\.\pipe\invidtui-socket`
}
