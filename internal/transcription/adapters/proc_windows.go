//go:build windows

package adapters

import "os/exec"

func killProcessGroupOnCancel(cmd *exec.Cmd) {}
