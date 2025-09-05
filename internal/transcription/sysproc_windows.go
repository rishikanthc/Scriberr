//go:build windows
// +build windows

package transcription

import "os/exec"

// configureCmdSysProcAttr is a no-op on Windows to keep builds portable.
// If full process tree termination is required, implement Windows-specific
// logic (e.g., using job objects) in the future.
func configureCmdSysProcAttr(cmd *exec.Cmd) {
    // No special attributes set on Windows here
}
