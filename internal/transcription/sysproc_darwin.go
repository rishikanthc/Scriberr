//go:build darwin
// +build darwin

package transcription

import (
    "os/exec"
    "syscall"
)

// configureCmdSysProcAttr sets process group on macOS so we can kill children.
func configureCmdSysProcAttr(cmd *exec.Cmd) {
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

