//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package agent

import (
	"fmt"
	"os/exec"
	"runtime"
)

func preparePluginScriptProcess(*exec.Cmd) (func(), error) {
	return nil, fmt.Errorf("plugin script execution is not supported on %s until process-tree cancellation is implemented", runtime.GOOS)
}
