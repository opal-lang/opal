//go:build !windows

package decorator

import (
	"os/exec"
	"syscall"
)

func configureCommandForCancellation(cmd *exec.Cmd) {
	// Create a dedicated process group so cancellation can terminate children.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateCommandOnCancel(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// Send SIGKILL to the process group (negative pid) to terminate parent+children.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
