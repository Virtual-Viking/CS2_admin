//go:build windows

package instance

import (
	"os/exec"
	"syscall"
)

// hideWindow sets SysProcAttr so the command runs without a visible console window.
// Uses only CREATE_NO_WINDOW to prevent the console host from creating a window.
// HideWindow is NOT used because it can conflict with CREATE_NO_WINDOW for GUI-subsystem apps.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
