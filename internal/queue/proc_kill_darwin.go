//go:build darwin
// +build darwin

package queue

import (
	"os"
	"syscall"
)

// killProcessTree sends SIGKILL to the entire process group on macOS.
func killProcessTree(p *os.Process) error {
	return syscall.Kill(-p.Pid, syscall.SIGKILL)
}
