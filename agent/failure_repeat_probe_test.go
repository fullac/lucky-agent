package agent

import (
	"encoding/json"
	"testing"

	"github.com/yurika0211/luckyagent-runtime/provider"
)

func mustToolArgs(t *testing.T, command string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"command": command})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestIssue315SessionShapeStopsOnFourthVariedDenial(t *testing.T) {
	state := &streamConvergenceState{repeatToolCallLimit: 150, toolOnlyIterationLimit: 150}
	denial := "access denied: command references restricted path (~/.luckyagent/config.json)"
	commands := []string{
		mustToolArgs(t, "cat ~/.luckyagent/config.json"),
		mustToolArgs(t, "head -n 40 ~/.luckyagent/config.json"),
		mustToolArgs(t, "sed -n 1,20p ~/.luckyagent/config.json"),
		mustToolArgs(t, `python3 -c "print(open('/home/user/.luckyagent/config.json').read())"`),
	}
	for i, args := range commands[:3] {
		content := ""
		if i == 0 {
			content = "still looking"
		}
		stopped, _ := state.trackToolCallPattern([]provider.ToolCall{{Name: "terminal", Arguments: args}}, content)
		if stopped {
			t.Fatalf("stopped on attempt %d", i+1)
		}
		state.rememberToolCallResult("terminal", args, denial, 0)
	}
	stopped, sigs := state.trackToolCallPattern([]provider.ToolCall{{Name: "terminal", Arguments: commands[3]}}, "")
	if !stopped {
		t.Fatalf("fourth varied denial was not stopped, parsed=%q sigs=%v counts=%v", toolCallPathArg(commands[3]), sigs, state.failureRepeatCount)
	}
	fresh := &streamConvergenceState{repeatToolCallLimit: 150, toolOnlyIterationLimit: 150}
	for i := 0; i < 5; i++ {
		args := `{"command":"rg -n duration_ms agent/loop.go"}`
		stopped, _ := fresh.trackToolCallPattern([]provider.ToolCall{{Name: "terminal", Arguments: args}}, "")
		if stopped {
			t.Fatalf("successful varied inspection stopped on round %d", i)
		}
		fresh.rememberToolCallResult("terminal", args, "873: duration_ms", 0)
	}
}
