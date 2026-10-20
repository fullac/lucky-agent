package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/yurika0211/luckyagent-runtime/provider"
)

func TestToolResultStatus(t *testing.T) {
	cases := []struct {
		result string
		status string
	}{
		{"ok", "ok"},
		{"Error: command timed out after 120 seconds", "timeout"},
		{"access denied: command references restricted path (config.json)", "sandbox"},
		{"Error: path /tmp/out is outside sandbox", "sandbox"},
		{"sudo: a password is required", "sudo"},
		{"sudo: a terminal is required to read the password", "sudo"},
		{"sh: 1: rg: not found in $PATH", "missing"},
		{"Error: pdftotext is not installed", "missing"},
		{"Error: boom", "error"},
		{"failed [exit code: 2]", "error"},
	}
	for _, tc := range cases {
		if got := toolResultStatus(tc.result); got != tc.status {
			t.Fatalf("toolResultStatus(%q) = %q, want %q", tc.result, got, tc.status)
		}
	}
}

func TestObservedToolMessageKeepsTimingOffTheModelContext(t *testing.T) {
	msg := observedToolMessage("run-1", executedToolCall{
		ToolCall: provider.ToolCall{ID: "call-1", Name: "terminal"},
		Result:   "Error: command timed out after 30 seconds",
		Duration: 30 * time.Second,
	}, "Error: command timed out after 30 seconds")
	if msg.Role != "tool" || msg.ToolCallID != "call-1" || msg.Name != "terminal" {
		t.Fatalf("identity = %#v", msg)
	}
	if msg.RunID != "run-1" || msg.DurationMS != 30000 || msg.Status != "timeout" || msg.StartedAt == nil {
		t.Fatalf("observation = %#v", msg)
	}
	if msg.StartedAt.After(time.Now()) || time.Since(*msg.StartedAt) < 20*time.Second {
		t.Fatalf("started_at = %v", msg.StartedAt)
	}
	if strings.Contains(msg.Content, "run-1") {
		t.Fatalf("run id leaked into model content: %s", msg.Content)
	}
}
