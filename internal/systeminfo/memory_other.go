//go:build !linux && !darwin && !windows

package systeminfo

func TotalMemoryBytes() (uint64, error) {
	return 0, nil
}
