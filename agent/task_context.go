package agent

import (
	"encoding/json"
	"strings"

	"github.com/yurika0211/lucky-agent/agent/artifact"
	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-runtime/turn"
)

// Reuse the turn's existing intent classifier. A declared plan is never proof
// that an execution request was completed, even before the first tool call.
func (a *Agent) needsExecutionReview(goal, candidate string) bool {
	allowed, recognized := a.intentAllowedTools(goal)
	if recognized && len(allowed) > 0 {
		return true
	}
	text := strings.ToLower(strings.TrimSpace(goal))
	if artifact.HasIntent(goal) || intentTextContainsAny(text, "执行", "运行", "优化", "实现", "完成任务", "部署", "implement ", "edit ", "fix ", "run ", "optimize", "refactor") {
		return true
	}
	answer := strings.ToLower(strings.TrimSpace(candidate))
	return intentTextContainsAny(answer, "i will ", "i'll ", "我会", "我将", "接下来我", "准备开始")
}

// This compact ledger is derived from persisted operations, not inferred from
// assistant prose. Operation completion and task acceptance remain distinct.
func durableTaskState(task *turn.DurableTask, cp *durableCheckpoint) provider.Message {
	type operation struct {
		ID     string `json:"id"`
		Tool   string `json:"tool"`
		Status string `json:"status"`
	}
	completed, failed := 0, 0
	var recent []operation
	for _, op := range task.Operations {
		if op.State == "completed" {
			if op.Failed {
				failed++
			} else {
				completed++
			}
		}
		status := op.State
		if op.Failed {
			status = "failed"
		}
		recent = append(recent, operation{op.ID, op.Name, status})
	}
	if len(recent) > 8 {
		recent = recent[len(recent)-8:]
	}
	var pending []string
	for _, op := range cp.Pending {
		pending = append(pending, op.ID)
	}
	next := "Continue the current task; collect evidence for unmet acceptance criteria."
	if len(pending) > 0 {
		next = "Resolve recorded pending operations before starting new work."
	} else if cp.Candidate != "" {
		next = "Verify the candidate against the original goal and acceptance criteria."
	}
	data, _ := json.Marshal(map[string]any{"task_id": task.ID, "successful_operations": completed, "failed_operations": failed, "recent_operations": recent, "pending_operations": pending, "task_verified": false, "next_action": next})
	return provider.Message{Role: "system", Content: "[Task State]\nRecorded execution facts, not a completion claim. Do not repeat completed side effects.\n" + string(data)}
}
