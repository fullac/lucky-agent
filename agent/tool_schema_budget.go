package agent

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
)

// Select complete schemas, preserving explicitly requested/forced tools and
// the current tool round. Optional schemas use at most one third of the input
// window; required schemas may borrow more while pinned context still fits.
func (a *Agent) selectRequestTools(b *modelRequestBudget, messages []provider.Message, priorities []contextx.Message, opts provider.CallOptions, turnRequired []string) (provider.CallOptions, error) {
	if b == nil || len(opts.Tools) == 0 {
		return opts, nil
	}
	var query strings.Builder
	for i, msg := range messages {
		if msg.Role == "system" && strings.HasPrefix(msg.Content, "[Current Task]") {
			query.WriteString(msg.Content)
			query.WriteByte('\n')
		}
		if priorities[i].Priority == contextx.PriorityCritical && msg.Role == "user" && !isTransientComputerObservation(msg) {
			query.WriteString(msg.Content)
			for _, part := range msg.ContentParts {
				if part.Type == "text" {
					query.WriteString(part.Text)
				}
			}
		}
	}
	text := query.String()
	required := makeToolNameSet(a.extractRequiredToolNames(text))
	if required == nil {
		required = map[string]struct{}{}
	}
	for _, name := range turnRequired {
		required[name] = struct{}{}
	}
	if forced := forcedToolChoiceName(opts.ToolChoice); forced != "" {
		required[forced] = struct{}{}
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if len(messages[i].ToolCalls) > 0 {
			for _, call := range messages[i].ToolCalls {
				required[call.Name] = struct{}{}
			}
			break
		}
	}
	if len(opts.Tools) == 1 {
		required[functionToolNameFromSchema(opts.Tools[0])] = struct{}{}
	}
	preferred, _ := a.intentAllowedTools(text)
	for _, name := range []string{"file_read", "file_list", "terminal", "web_search", "web_fetch", "ask_user", "context_history_read", "skill_read"} {
		if preferred == nil {
			preferred = map[string]struct{}{}
		}
		preferred[name] = struct{}{}
	}
	var pinned []provider.Message
	for i, priority := range priorities {
		if priority.Priority == contextx.PriorityCritical {
			pinned = append(pinned, messages[i])
		}
	}
	available := b.limit - estimateProviderRequest(b.est, pinned, nil)
	if available <= 0 {
		return opts, fmt.Errorf("current task and pinned instructions exceed context input budget %d", b.limit)
	}
	limit := b.limit / 3
	if limit > available {
		limit = available
	}
	type entry struct {
		index, cost, score int
		required           bool
	}
	entries := make([]entry, 0, len(opts.Tools))
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	mandatory := 0
	for i, schema := range opts.Tools {
		name := functionToolNameFromSchema(schema)
		_, must := required[name]
		cost := estimateToolSchemas(b.est, []map[string]any{schema})
		score := 0
		if _, ok := preferred[name]; ok {
			score += 50
		}
		fn, _ := schema["function"].(map[string]any)
		description, _ := fn["description"].(string)
		haystack := strings.ToLower(name + " " + description)
		for _, word := range words {
			if len([]rune(word)) >= 3 && strings.Contains(haystack, word) {
				score++
			}
		}
		if must {
			mandatory += cost
		}
		entries = append(entries, entry{index: i, cost: cost, score: score, required: must})
	}
	if mandatory > available {
		return opts, fmt.Errorf("required tool schemas need about %d tokens but only %d remain in context input budget %d", mandatory, available, b.limit)
	}
	if mandatory > limit {
		limit = mandatory
	}
	if estimateToolSchemas(b.est, opts.Tools) <= limit {
		return opts, nil
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].required != entries[j].required {
			return entries[i].required
		}
		return entries[i].score > entries[j].score
	})
	selected := make(map[int]bool)
	used := 0
	for _, item := range entries {
		if item.required || used+item.cost <= limit {
			selected[item.index] = true
			used += item.cost
		}
	}
	if len(selected) == 0 {
		return opts, fmt.Errorf("no tool schema fits the %d-token schema allocation within context input budget %d", limit, b.limit)
	}
	tools := make([]map[string]any, 0, len(selected))
	for i, schema := range opts.Tools {
		if selected[i] {
			tools = append(tools, schema)
		}
	}
	opts.Tools = tools
	opts.ToolChoice = normalizeToolChoiceForTools(opts.ToolChoice, tools)
	return opts, nil
}
