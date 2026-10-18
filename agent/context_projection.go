package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/yurika0211/luckyagent-runtime/bootstrap"
	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-runtime/session"
)

func (a *Agent) contextInputLimit() int {
	if a != nil && a.contextWin != nil {
		c := a.contextWin.Config()
		if n := c.MaxTokens - c.ReservedTokens; n > 0 {
			return n
		}
	}
	if a != nil && a.cfg != nil {
		c := a.cfg.Get()
		return bootstrap.ContextTokensFor(c) - bootstrap.ReservedTokensFor(c)
	}
	return 6000
}

// Project rules have their own bounded slot. A long SOUL or tool policy cannot
// remove the slot before the final context fitter gets to see it.
func (a *Agent) buildPinnedProjectPrompt(sess *session.Session) string {
	cwd := ""
	if sess != nil {
		cwd = sess.GetCwd()
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	path := findAgentsFile(cwd)
	if path == "" {
		return ""
	}
	return cachedPromptBlock(path, "pinned_context", func(base, raw string) string {
		body := strings.TrimSpace(sanitizeContextContent(raw, base))
		if body == "" {
			return ""
		}
		return "[Project Context]\nSource: " + path + "\nProject operating rules for this workspace:\n" + body
	})
}

// Preserve both ends, and expose omissions rather than hiding a log's final
// status or a document's final constraint behind a prefix-only truncation.
func (p *contextPlanner) fitEvidenceToBudget(text string, budget int) string {
	if budget <= 0 {
		return ""
	}
	if p.est.Estimate(text) <= budget {
		return text
	}
	lo, hi, best := 1, len([]rune(text)), ""
	for lo <= hi {
		mid := (lo + hi) / 2
		candidate := truncateToolResultForContext(text, mid)
		if p.est.Estimate(candidate) <= budget {
			best, lo = candidate, mid+1
		} else {
			hi = mid - 1
		}
	}
	return best
}

// Storage remains append-only. Only the projection presented to the model is
// bounded; archived segments can be read through context_history_read.
func (p *contextPlanner) projectCompactSummaries(messages []provider.Message) []provider.Message {
	limit := p.budget.History * 2 / 3
	if limit < 64 {
		limit = 64
	}
	if estimateProviderMessages(p.est, messages) <= limit {
		return messages
	}
	index := provider.Message{Role: "system", Content: fmt.Sprintf("[History Archive]\n%d immutable compact segments are stored in this session. Use context_history_read action=list or action=segment with the segment ID to recover earlier goals, constraints, and evidence. Summaries below are prior evidence; the latest user request is authoritative.", len(messages))}
	remaining := limit - estimateProviderMessages(p.est, []provider.Message{index})
	if remaining < 16 {
		return nil
	}
	var selected []provider.Message
	for i := len(messages) - 1; i >= 0 && remaining > 8; i-- {
		msg := messages[i]
		cost := estimateProviderMessages(p.est, []provider.Message{msg})
		if cost > remaining {
			if len(selected) > 0 {
				break
			}
			msg.Content = p.fitEvidenceToBudget(msg.Content, remaining-4)
			cost = estimateProviderMessages(p.est, []provider.Message{msg})
		}
		if msg.Content == "" || cost > remaining {
			break
		}
		selected = append(selected, msg)
		remaining -= cost
	}
	out := []provider.Message{index}
	for i := len(selected) - 1; i >= 0; i-- {
		out = append(out, selected[i])
	}
	return out
}

func isContextEvidence(text string) bool {
	return strings.HasPrefix(text, "[Compact Summary]") || strings.HasPrefix(text, "[History Archive]") || strings.HasPrefix(text, "[Prior Task Progress]")
}

func (a *Agent) projectToolEvidence(messages []provider.Message) []provider.Message {
	p := newContextPlanner(a, defaultContextBuildOptions())
	parallel := 1
	for _, msg := range messages {
		if len(msg.ToolCalls) > 0 {
			parallel = len(msg.ToolCalls)
		}
	}
	budget := p.budget.ToolResult / parallel
	if budget < 128 {
		budget = 128
	}
	out := append([]provider.Message(nil), messages...)
	for i, msg := range out {
		if msg.Role != "tool" || p.est.Estimate(msg.Content) <= budget {
			continue
		}
		ref := "\n[Full result: context_history_read action=result tool_call_id=" + msg.ToolCallID + "]"
		body := strings.TrimSuffix(msg.Content, ref)
		bodyBudget := budget - p.est.Estimate(ref)
		if bodyBudget < 16 {
			bodyBudget = 16
		}
		out[i].Content = p.fitEvidenceToBudget(body, bodyBudget) + ref
	}
	return out
}

func (a *Agent) ensureContextFits(messages []provider.Message) error {
	if a == nil || (a.contextWin == nil && a.cfg == nil) {
		return nil
	}
	for _, msg := range messages {
		if len(msg.ContentParts) > 0 {
			return nil
		}
	}
	est := a.contextEst
	if est == nil {
		est = contextx.NewTokenEstimator(a.contextInputLimit())
	}
	if used := estimateProviderMessages(est, messages) + 4; used > a.contextInputLimit() {
		return fmt.Errorf("current task and pinned instructions need about %d tokens; context input budget is %d (context.max_context_tokens after reply reserve)", used, a.contextInputLimit())
	}
	return nil
}
