package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yurika0211/luckyagent-runtime/session"
	"github.com/yurika0211/luckyagent-runtime/tool"
)

// Session identity comes from the host execution context, never model arguments.
func (a *Agent) registerContextHistoryTool() {
	a.tools.Register(&tool.Tool{Name: "context_history_read", Category: tool.CatBuiltin, Enabled: true, Permission: tool.PermAuto,
		Description: "Read archived compact segments or full tool results from the current conversation only. Use paginated offsets to recover evidence omitted from the model window.",
		Parameters: map[string]tool.Param{
			"action":       {Type: "string", Required: true, Description: "list, segment, or result"},
			"segment_id":   {Type: "string", Description: "ID from action=list or a compact summary"},
			"tool_call_id": {Type: "string", Description: "Call ID from the tool result reference"},
			"offset":       {Type: "integer", Description: "Zero-based character offset, or segment offset for list"},
			"limit":        {Type: "integer", Description: "Maximum 6000 characters, or 50 segments for list"},
		},
		ContextDetailedHandler: func(exec tool.ExecutionContext, args map[string]any) (tool.ToolCallResult, error) {
			if a.sessions == nil || strings.TrimSpace(exec.SessionID) == "" {
				return tool.ToolCallResult{}, fmt.Errorf("context_history_read requires a host-bound session")
			}
			sess, ok := a.sessions.Get(exec.SessionID)
			if !ok {
				return tool.ToolCallResult{}, fmt.Errorf("current session not found")
			}
			body, err := readContextHistory(sess, args)
			return tool.ToolCallResult{Output: body}, err
		},
		Handler: func(map[string]any) (string, error) {
			return "", fmt.Errorf("context_history_read requires a context-aware host call")
		},
	})
}

func historyPageNumber(args map[string]any, name string, fallback, max int) int {
	v := fallback
	switch n := args[name].(type) {
	case float64:
		v = int(n)
	case int:
		v = n
	}
	if v < 0 {
		v = 0
	}
	if v > max {
		v = max
	}
	return v
}

func readContextHistory(sess *session.Session, args map[string]any) (string, error) {
	action, _ := args["action"].(string)
	all := sess.GetMessages()
	segments, _, _ := session.CompactSegments(all)
	offset := historyPageNumber(args, "offset", 0, 1<<30)
	var content, source string
	switch action {
	case "list":
		limit := historyPageNumber(args, "limit", 20, 50)
		if limit < 1 {
			limit = 20
		}
		type entry struct {
			ID       string `json:"id"`
			From, To int
		}
		entries := make([]entry, 0, limit)
		for i := offset; i < len(segments) && len(entries) < limit; i++ {
			s := segments[i]
			entries = append(entries, entry{s.ID, s.FromMessage, s.ToMessage})
		}
		next := offset + len(entries)
		if next >= len(segments) {
			next = -1
		}
		body, err := json.Marshal(map[string]any{"segments": entries, "total": len(segments), "next_offset": next})
		return string(body), err
	case "segment":
		id, _ := args["segment_id"].(string)
		for _, s := range segments {
			if s.ID == id && id != "" {
				content, source = s.Summary, id
				break
			}
		}
	case "result":
		id, _ := args["tool_call_id"].(string)
		for i := len(all) - 1; i >= 0; i-- {
			if all[i].Role == "tool" && all[i].ToolCallID == id && id != "" {
				var err error
				content, err = sess.ResolveMessageContent(all[i])
				if err != nil {
					return "", err
				}
				source = id
				break
			}
		}
	default:
		return "", fmt.Errorf("action must be list, segment, or result")
	}
	if source == "" {
		return "", fmt.Errorf("requested evidence was not found in the current session")
	}
	runes := []rune(content)
	if offset > len(runes) {
		offset = len(runes)
	}
	limit := historyPageNumber(args, "limit", 4000, 6000)
	if limit < 1 {
		limit = 4000
	}
	end := offset + limit
	if end > len(runes) {
		end = len(runes)
	}
	next := end
	if end == len(runes) {
		next = -1
	}
	body, err := json.Marshal(map[string]any{"source": source, "content": string(runes[offset:end]), "offset": offset, "total_chars": len(runes), "next_offset": next, "evidence_only": true})
	return string(body), err
}
