//go:build windows

package steam

import (
	"os/exec"
	"syscall"
)

// hideWindow sets SysProcAttr so the command runs without a visible console window.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
