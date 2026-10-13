package agent

import "testing"

func TestExplicitReadSurvivesNegatedWrite(t *testing.T) {
	prompt := "严格按顺序做，不要解释。\n第一轮只调用一次 file_read，读取 internal/gateway/telegram/handler.go。\n不要调用 file_write、ask_user、subagent。"
	a := agentWithIntentGateTools(t)
	t.Setenv(toolIntentGatingEnv, "")
	loopCfg := DefaultLoopConfig()
	sanitizeLoopConfig(&loopCfg)

	a.applyIntentToolGating(&loopCfg, prompt)
	opts := a.buildLoopCallOptions(prompt, loopCfg)
	visible := toolNamesFromSchemas(opts.Tools)

	assertEnabledTools(t, visible, "file_read")
	if len(visible) < 2 {
		t.Fatalf("negated file_write should not replace the schema, got %v", visible)
	}
}
