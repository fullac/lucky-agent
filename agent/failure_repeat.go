package agent

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

const failureRepeatLimit = 3

func failureRepeatKey(name, result string) string {
	name = strings.TrimSpace(name)
	kind, target := failureRepeatParts(result)
	if name == "" || kind == "" || target == "" {
		return ""
	}
	return name + "|" + kind + "|" + target
}

func failureRepeatParts(result string) (kind, target string) {
	text := strings.TrimSpace(result)
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "restricted path") || strings.Contains(lower, "outside sandbox") || strings.Contains(lower, "access denied"):
		kind = "sandbox"
	case strings.Contains(lower, "command timed out") || strings.Contains(lower, "timed out after"):
		kind = "timeout"
	case strings.Contains(lower, "no such file") || strings.Contains(lower, "not found") || strings.Contains(lower, "cannot find"):
		kind = "missing"
	default:
		return "", ""
	}
	target = failureRepeatTarget(text)
	if target == "" {
		target = failureRepeatTarget(firstLine(text))
	}
	return kind, target
}

func failureRepeatTarget(text string) string {
	for _, marker := range []string{"Requested: ", "restricted path (", "restricted path "} {
		if idx := strings.Index(text, marker); idx >= 0 {
			rest := strings.TrimSpace(text[idx+len(marker):])
			rest = strings.Trim(rest, " )]")
			if path := cleanFailurePath(rest); path != "" {
				return path
			}
		}
	}
	fields := strings.Fields(text)
	for i := len(fields) - 1; i >= 0; i-- {
		field := strings.Trim(fields[i], "`,;:()[]'\"")
		if path := cleanFailurePath(field); path != "" && looksLikeFailurePath(path) {
			return path
		}
	}
	if idx := strings.LastIndex(text, ": "); idx >= 0 {
		return cleanFailurePath(text[:idx])
	}
	return ""
}

func looksLikeFailurePath(path string) bool {
	return strings.Contains(path, "/") || strings.Contains(path, `\`) || strings.HasPrefix(path, "~") || strings.Contains(path, ".")
}

func cleanFailurePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "`,;:()[]'\"")
	if path == "" || strings.ContainsAny(path, " \t\n") {
		return ""
	}
	path = strings.TrimPrefix(path, "file://")
	slash := filepath.ToSlash(path)
	switch {
	case strings.HasPrefix(slash, "~/"):
		slash = slash[2:]
	case strings.HasPrefix(slash, "/home/") || strings.HasPrefix(slash, "/Users/"):
		parts := strings.SplitN(strings.TrimPrefix(slash, "/"), "/", 3)
		if len(parts) == 3 {
			slash = parts[2]
		}
	}
	slash = filepath.Clean(slash)
	slash = filepath.ToSlash(slash)
	slash = strings.TrimPrefix(slash, "./")
	return strings.ToLower(slash)
}

func firstLine(text string) string {
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		return text[:idx]
	}
	return text
}

func (s *streamConvergenceState) failureRepeated(name, arguments string) bool {
	if s == nil {
		return false
	}
	return failureCountReached(s.failureRepeatCount, name, arguments)
}

func (s *streamConvergenceState) noteFailureRepeat(name, result string) {
	if s == nil {
		return
	}
	key := failureRepeatKey(name, result)
	if key == "" {
		return
	}
	if s.failureRepeatCount == nil {
		s.failureRepeatCount = make(map[string]int)
	}
	s.failureRepeatCount[key]++
}

func (s *loopRuntimeState) failureRepeated(name, arguments string) bool {
	if s == nil {
		return false
	}
	return failureCountReached(s.failureRepeatCount, name, arguments)
}

func (s *loopRuntimeState) noteFailureRepeat(name, result string) {
	if s == nil {
		return
	}
	key := failureRepeatKey(name, result)
	if key == "" {
		return
	}
	if s.failureRepeatCount == nil {
		s.failureRepeatCount = make(map[string]int)
	}
	s.failureRepeatCount[key]++
}

func failureCountReached(counts map[string]int, name, arguments string) bool {
	path := toolCallPathArg(arguments)
	if path == "" || counts == nil {
		return false
	}
	for _, kind := range []string{"sandbox", "timeout", "missing"} {
		if counts[name+"|"+kind+"|"+path] >= failureRepeatLimit {
			return true
		}
	}
	return false
}

func toolCallPathArg(arguments string) string {
	var args map[string]any
	if json.Unmarshal([]byte(arguments), &args) != nil {
		return ""
	}
	for _, key := range []string{"path", "file", "workdir"} {
		value, _ := args[key].(string)
		if path := cleanFailurePath(value); path != "" && looksLikeFailurePath(path) {
			return path
		}
	}
	command, _ := args["command"].(string)
	return failureRepeatTarget(command)
}
