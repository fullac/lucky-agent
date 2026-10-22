package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yurika0211/luckyagent-runtime/tool"
)

const (
	ScriptTimeout     = 60 * time.Second
	ScriptOutputLimit = 64 << 10
	ScriptWaitDelay   = 250 * time.Millisecond
)

var argumentName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// SelectScript returns the regular script file for one skill tool. An empty
// path means the skill has no scripts directory. Injected handlers and symlink
// script directories are rejected.
func SelectScript(skillDir string, definition tool.SkillToolDef) (string, error) {
	if definition.Handler != nil {
		return "", fmt.Errorf("injected plugin skill handlers are unsupported")
	}
	root, err := filepath.Abs(skillDir)
	if err != nil {
		return "", err
	}
	scriptsDir := filepath.Join(root, "scripts")
	info, err := os.Lstat(scriptsDir)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("skill scripts must be a regular directory")
	}
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		return "", err
	}
	var scripts []string
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("plugin script %q is a symlink", entry.Name())
		}
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".sh" && ext != ".py" && ext != ".js" {
			continue
		}
		if !entry.Type().IsRegular() {
			return "", fmt.Errorf("plugin script %q is not a regular file", entry.Name())
		}
		scripts = append(scripts, filepath.Join(scriptsDir, entry.Name()))
	}
	// Command contains already parsed CLI subcommand tokens. A single CLI
	// script receives those tokens directly, never through a shell string.
	if len(definition.Command) > 0 && len(scripts) == 1 {
		return scripts[0], nil
	}
	if definition.Name == "" || strings.ContainsAny(definition.Name, "/\\") || definition.Name == ".." {
		return "", fmt.Errorf("invalid plugin skill tool name %q", definition.Name)
	}
	for _, ext := range []string{".sh", ".py", ".js"} {
		candidate := filepath.Join(scriptsDir, definition.Name+ext)
		for _, script := range scripts {
			if script == candidate {
				return script, nil
			}
		}
	}
	if len(scripts) == 1 {
		return scripts[0], nil
	}
	if len(definition.Command) > 0 {
		return "", fmt.Errorf("CLI subcommands require an unambiguous plugin script")
	}
	return "", nil
}

// Command builds the interpreter argv for one selected script.
func Command(script string, arguments []string) ([]string, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("plugin script execution is not supported on Windows until process-tree cancellation is implemented")
	}
	var binary string
	switch strings.ToLower(filepath.Ext(script)) {
	case ".sh":
		binary = "sh"
	case ".py":
		binary = "python3"
	case ".js":
		binary = "node"
	default:
		return nil, fmt.Errorf("unsupported plugin script type")
	}
	return append([]string{binary, script}, arguments...), nil
}

// Environment copies the process environment and adds SKILL_ARG_* values.
func Environment(args map[string]any) ([]string, error) {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	for key, value := range args {
		// Host-only injection keys must not become plugin arguments.
		if strings.HasPrefix(key, "_") {
			continue
		}
		if !argumentName.MatchString(key) {
			return nil, fmt.Errorf("invalid plugin argument name %q", key)
		}
		var encoded string
		if text, ok := value.(string); ok {
			encoded = text
		} else {
			data, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("plugin argument %s: %w", key, err)
			}
			encoded = string(data)
		}
		if strings.ContainsRune(encoded, 0) {
			return nil, fmt.Errorf("plugin argument %s contains NUL", key)
		}
		values["SKILL_ARG_"+strings.ToUpper(key)] = encoded
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	env := make([]string, 0, len(values))
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}
	return env, nil
}

// BoundedOutput keeps the first limit bytes and records truncation.
type BoundedOutput struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	Limit     int
	truncated bool
}

func (output *BoundedOutput) Write(data []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()
	n := len(data)
	remaining := output.Limit - output.buffer.Len()
	if remaining < 0 {
		remaining = 0
	}
	if len(data) > remaining {
		data = data[:remaining]
		output.truncated = true
	}
	_, _ = output.buffer.Write(data)
	return n, nil
}

func (output *BoundedOutput) String() string {
	output.mu.Lock()
	defer output.mu.Unlock()
	text := output.buffer.String()
	if output.truncated {
		text += "\n[Plugin output truncated at 65536 bytes]"
	}
	return text
}
