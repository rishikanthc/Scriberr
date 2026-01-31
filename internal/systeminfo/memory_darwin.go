//go:build darwin

package systeminfo

import "golang.org/x/sys/unix"

func TotalMemoryBytes() (uint64, error) {
	return unix.SysctlUint64("hw.memsize")
}
