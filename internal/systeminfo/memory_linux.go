//go:build linux

package systeminfo

import "golang.org/x/sys/unix"

func TotalMemoryBytes() (uint64, error) {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		return 0, err
	}
	return info.Totalram * uint64(info.Unit), nil
}
