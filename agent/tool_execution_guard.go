package agent

import (
	"fmt"

	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-tools/execpolicy"
)

// toolExecutionGuard adapts provider calls to the shared turn policy.
type toolExecutionGuard struct {
	core          *execpolicy.Guard
	restrictTools bool
	allowedTools  map[string]struct{}
}

func (g *toolExecutionGuard) blockMessage(call provider.ToolCall) (string, bool) {
	if g == nil {
		return "", false
	}
	if g.restrictTools && !toolNameInSet(g.allowedTools, call.Name) {
		return fmt.Sprintf("Tool %q is outside this role's tool allowlist.", call.Name), true
	}
	return g.core.BlockMessage(execpolicy.Call{Name: call.Name, Arguments: call.Arguments})
}

var parseToolCallArgs = execpolicy.ParseArgs
var guardStringArg = execpolicy.StringArg
