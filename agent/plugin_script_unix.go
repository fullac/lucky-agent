//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package agent

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// One process group belongs to one invocation. CommandContext's default Kill
// only stops the interpreter, leaving its children alive; WaitDelay alone only
// closes their inherited output pipes. Kill the group both on cancellation and
// after the interpreter exits, so a background child cannot outlive the call.
func preparePluginScriptProcess(command *exec.Cmd) (func(), error) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	killGroup := func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.Cancel = killGroup
	return func() { _ = killGroup() }, nil
}
