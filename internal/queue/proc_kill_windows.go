//go:build windows
// +build windows

package queue

import "os"

// killProcessTree attempts to kill the process. Windows lacks a simple
// process group SIGKILL equivalent; callers may need a more robust tree kill.
func killProcessTree(p *os.Process) error {
	return p.Kill()
}
