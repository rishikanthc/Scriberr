//go:build linux
// +build linux

package transcription

import (
	"os/exec"
	"syscall"
)

// configureCmdSysProcAttr sets process group on Linux so we can kill children.
func configureCmdSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
