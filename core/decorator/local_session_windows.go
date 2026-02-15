//go:build windows

package decorator

import "os/exec"

func configureCommandForCancellation(_ *exec.Cmd) {
	// Windows does not support Unix Setpgid process groups.
}

func terminateCommandOnCancel(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	_ = cmd.Process.Kill()
}
