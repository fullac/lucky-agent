// Package compact adapts provider messages to the shared compact engine.
package compact

import (
	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-runtime/session"
	compactlib "github.com/yurika0211/luckyagent-tools/compact"
)

const SummaryTimeout = compactlib.SummaryTimeout

// TextOptions carries the host summary wording. SummarizeTool stays with the
// caller because tool-result shortening is owned by the agent context planner.
func TextOptions(summarize func(name, result string) string) compactlib.TextOptions {
	return compactlib.TextOptions{
		SummarizeTool:     summarize,
		FallbackGoal:      "Continue the current LuckyAgent session from the available prior messages.",
		MergeFallbackGoal: "Continue the current LuckyAgent session from prior segment summaries.",
	}
}

// Messages copies provider messages into the compact engine shape.
func Messages(messages []provider.Message) []compactlib.Message {
	out := make([]compactlib.Message, len(messages))
	for i, message := range messages {
		calls := make([]compactlib.ToolCall, len(message.ToolCalls))
		for j, call := range message.ToolCalls {
			calls[j] = compactlib.ToolCall{Name: call.Name}
		}
		out[i] = compactlib.Message{Role: message.Role, Content: message.Content, ReasoningContent: message.ReasoningContent, Name: message.Name, ToolCalls: calls, CompactBoundary: session.IsCompactBoundary(message)}
	}
	return out
}

// Estimator adapts the runtime token estimator. A nil estimator stays nil.
func Estimator(est *contextx.TokenEstimator) compactlib.Estimator {
	if est == nil {
		return nil
	}
	return est
}
