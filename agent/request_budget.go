package agent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
)

// modelRequestBudget is an immutable policy for one provider request. It has no
// dependency on Agent resources, storage, or backend construction.
type modelRequestBudget struct {
	limit int
	est   *contextx.TokenEstimator
}

func estimateToolSchemas(est *contextx.TokenEstimator, tools []map[string]any) int {
	if est == nil {
		return 0
	}
	total := 0
	for _, schema := range tools {
		encoded, err := json.Marshal(schema)
		if err != nil {
			// Invalid schemas are rejected before dispatch; retain an estimate
			// for diagnostics without ignoring their cost.
			return int(^uint(0)>>1) / 4
		}
		total += est.Estimate(string(encoded)) + 8
	}
	return total
}

func estimateProviderRequest(est *contextx.TokenEstimator, messages []provider.Message, tools []map[string]any) int {
	if est == nil {
		return 0
	}
	total := estimateToolSchemas(est, tools) + 4
	for _, message := range messages {
		content := modelMessageContentTokens(est, message)
		message.Content = ""
		total += estimateProviderMessages(est, []provider.Message{message}) + content
	}
	return total
}

func modelMessageContentTokens(est *contextx.TokenEstimator, message provider.Message) int {
	plain := est.Estimate(message.Content)
	if len(message.ContentParts) == 0 {
		return plain
	}
	total := 0
	for _, part := range message.ContentParts {
		if part.Type == "text" {
			total += est.Estimate(part.Text)
		}
		if part.Type == "image" && part.Image != nil {
			// These are fixed allowances, not provider token billing. Text is
			// always checked even when the vision estimate is approximate.
			switch strings.ToLower(strings.TrimSpace(part.Image.Detail)) {
			case "low":
				total += 512
			case "high":
				total += 2048
			default:
				total += 1024
			}
		}
	}
	// Adapters can fall back to Content when all parts are empty, and some
	// providers only consume Content. Never let parts hide that text surface.
	if plain > total {
		return plain
	}
	return total
}

func (a *Agent) requestBudget() *modelRequestBudget {
	if a == nil || (a.contextWin == nil && a.cfg == nil) {
		return nil
	}
	limit := a.contextInputLimit()
	est := a.contextEst
	if est == nil {
		est = contextx.NewTokenEstimator(limit)
	}
	return &modelRequestBudget{limit: limit, est: est}
}

func (b *modelRequestBudget) check(messages []provider.Message, opts provider.CallOptions) error {
	if b == nil {
		return nil
	}
	if _, err := json.Marshal(opts.Tools); err != nil {
		return fmt.Errorf("invalid tool schema: %w", err)
	}
	schemaTokens := estimateToolSchemas(b.est, opts.Tools)
	if schemaTokens >= b.limit {
		return fmt.Errorf("tool schemas need about %d tokens; context input budget is %d; reduce the enabled tool set or schema descriptions", schemaTokens, b.limit)
	}
	used := estimateProviderRequest(b.est, messages, opts.Tools)
	if used > b.limit {
		return fmt.Errorf("complete model request needs about %d tokens (%d for tool schemas, with structured content included); context input budget is %d (context.max_context_tokens after reply reserve)", used, schemaTokens, b.limit)
	}
	return nil
}

// fit selects whole exchanges with the existing priority policy after reserving
// the actual schema cost. SourceID restores the original provider messages, so
// trimming cannot strip image parts, attachments, or other provider metadata.
func (b *modelRequestBudget) fit(messages []provider.Message, prioritized []contextx.Message, opts provider.CallOptions) []provider.Message {
	if b == nil || estimateProviderRequest(b.est, messages, opts.Tools) <= b.limit {
		return messages
	}
	schemas := estimateToolSchemas(b.est, opts.Tools)
	if schemas >= b.limit {
		return messages // check reports the schema overflow before dispatch
	}
	for i := range prioritized {
		prioritized[i].SourceID = strconv.Itoa(i)
		// Only the fitter sees the surrogate. It supplies the measured cost
		// of structured text/images without converting the original payload.
		cost := modelMessageContentTokens(b.est, messages[i])
		if cost > b.limit {
			cost = b.limit + 1
		}
		prioritized[i].Content = strings.Repeat("x", 4*cost)
	}
	window := contextx.NewContextWindow(contextx.WindowConfig{
		MaxTokens: b.limit, ReservedTokens: schemas, Strategy: contextx.TrimLowPriority,
	})
	fitted, _ := window.Fit(prioritized)
	out := make([]provider.Message, 0, len(fitted))
	for _, selected := range fitted {
		index, err := strconv.Atoi(selected.SourceID)
		if err == nil && index >= 0 && index < len(messages) {
			out = append(out, messages[index])
		}
	}
	return out
}

func (a *Agent) ensureRequestFits(messages []provider.Message, opts provider.CallOptions) error {
	return a.requestBudget().check(messages, opts)
}

func (a *Agent) prepareModelRequest(messages []provider.Message, opts provider.CallOptions, snapshot providerSnapshot) ([]provider.Message, provider.CallOptions, error) {
	// Schemas are charged only when the selected provider will receive them.
	if _, ok := snapshot.provider.(provider.FunctionCallingProvider); !ok {
		opts = provider.CallOptions{}
	}
	if _, err := json.Marshal(opts.Tools); err != nil {
		return nil, opts, fmt.Errorf("invalid tool schema: %w", err)
	}
	budget := snapshot.budget
	if budget == nil {
		budget = a.requestBudget()
	}
	if budget != nil {
		priorities := a.requestPriorities(messages)
		var err error
		opts, err = a.selectRequestTools(budget, messages, priorities, opts, snapshot.requiredTools)
		if err != nil {
			return nil, opts, err
		}
		messages = budget.fit(messages, priorities, opts)
	}
	return messages, opts, budget.check(messages, opts)
}

func (a *Agent) requestPriorities(messages []provider.Message) []contextx.Message {
	priorities := a.toContextMessages(messages)
	// Structured-only user input and transient observations both need a pinned
	// slot. The legacy text index cannot identify a user message with no Content.
	user, frame := false, false
	lastCall := -1
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "user" {
			if isTransientComputerObservation(msg) {
				if !frame {
					priorities[i].Priority = contextx.PriorityCritical
					frame = true
				}
			} else if !user {
				priorities[i].Priority = contextx.PriorityCritical
				user = true
			}
		}
		if lastCall < 0 && len(msg.ToolCalls) > 0 {
			lastCall = i
		}
	}
	if lastCall >= 0 {
		priorities[lastCall].Priority = contextx.PriorityCritical
		for i := lastCall + 1; i < len(messages) && messages[i].Role == "tool"; i++ {
			priorities[i].Priority = contextx.PriorityCritical
		}
	}
	return priorities
}
