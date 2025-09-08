//go:build linux
// +build linux

package queue

import (
	"os"
	"syscall"
)

// killProcessTree sends SIGKILL to the entire process group on Linux.
func killProcessTree(p *os.Process) error {
	return syscall.Kill(-p.Pid, syscall.SIGKILL)
}
