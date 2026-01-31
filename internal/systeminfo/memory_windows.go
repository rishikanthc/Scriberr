//go:build windows

package systeminfo

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func TotalMemoryBytes() (uint64, error) {
	var mem windows.MemStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	if err := windows.GlobalMemoryStatusEx(&mem); err != nil {
		return 0, err
	}
	return mem.TotalPhys, nil
}
