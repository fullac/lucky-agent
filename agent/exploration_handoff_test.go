package agent

import (
	"strings"
	"testing"

	"github.com/yurika0211/luckyagent-runtime/provider"
)

func TestShouldHandOffExplorationAfterRepeatedLargeReads(t *testing.T) {
	state := newLoopRuntimeState()
	state.consecutiveToolOnlyIters = 2
	state.explorationBytes = explorationHandoffBytes
	executed := []executedToolCall{{
		ToolCall: provider.ToolCall{ID: "1", Name: "file_read"},
		Result:   strings.Repeat("x", 100),
	}}

	if !shouldHandOffExploration(DefaultLoopConfig(), state, executed) {
		t.Fatal("expected exploration handoff")
	}
}

func TestShouldNotHandOffExplorationForEditsOrSubagents(t *testing.T) {
	state := newLoopRuntimeState()
	state.consecutiveToolOnlyIters = 2
	state.explorationBytes = explorationHandoffBytes
	edit := []executedToolCall{{
		ToolCall: provider.ToolCall{ID: "1", Name: "file_read"},
	}, {
		ToolCall: provider.ToolCall{ID: "2", Name: "file_write"},
	}}
	if shouldHandOffExploration(DefaultLoopConfig(), state, edit) {
		t.Fatal("file writes must stay in the parent context")
	}

	child := DefaultLoopConfig()
	child.Source = "subagent"
	if shouldHandOffExploration(child, state, []executedToolCall{{
		ToolCall: provider.ToolCall{ID: "1", Name: "file_read"},
	}}) {
		t.Fatal("sub-agent loops must not hand off again")
	}
}

func TestReplaceExplorationToolResultsKeepsOnlyAShortRecord(t *testing.T) {
	raw := strings.Repeat("log line\n", 2000)
	executed := []executedToolCall{{
		ToolCall: provider.ToolCall{ID: "call-1", Name: "file_read"},
		Result:   raw,
	}}
	messages := []provider.Message{{
		Role:       "tool",
		ToolCallID: "call-1",
		Name:       "file_read",
		Content:    raw,
	}}

	replaced := replaceExplorationToolResults(messages, executed)
	if strings.Contains(replaced[0].Content, "log line") {
		t.Fatal("raw tool output remained in parent context")
	}
	if !strings.Contains(replaced[0].Content, "exploration handoff") {
		t.Fatalf("replacement = %q", replaced[0].Content)
	}
	if !strings.Contains(replaced[0].Content, "16000 bytes") && !strings.Contains(replaced[0].Content, "bytes") {
		t.Fatalf("replacement did not record the removed size: %q", replaced[0].Content)
	}
}
