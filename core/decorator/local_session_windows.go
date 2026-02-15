//go:build windows

package decorator

import (
	"os/exec"
	"strconv"
)

func configureCommandForCancellation(_ *exec.Cmd) {
	// Windows does not support Unix Setpgid process groups.
}

func terminateCommandOnCancel(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// Try to terminate the full process tree first.
	// /T = child processes, /F = force terminate.
	pid := strconv.Itoa(cmd.Process.Pid)
	taskkill := exec.Command("taskkill", "/T", "/F", "/PID", pid)
	_ = taskkill.Run()

	// Fallback in case taskkill is unavailable or already exited.
	_ = cmd.Process.Kill()
}
