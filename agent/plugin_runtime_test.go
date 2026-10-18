package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	plugins "github.com/yurika0211/luckyagent-plugins"
	"github.com/yurika0211/luckyagent-runtime/config"
	"github.com/yurika0211/luckyagent-runtime/hook"
	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-runtime/sandbox"
	"github.com/yurika0211/luckyagent-runtime/session"
	"github.com/yurika0211/luckyagent-runtime/tool"
	"github.com/yurika0211/luckyagent-runtime/turn"
)

func pluginFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files[".lucky-plugin/plugin.json"] = `{"name":"fixture","version":"1.0.0"}`
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func pluginTestConfig(t *testing.T) *config.Manager {
	t.Helper()
	cfg, err := config.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Set("provider", "openai")
	cfg.Set("api_key", "sk-test")
	cfg.Set("model", "gpt-3.5-turbo")
	return cfg
}

func installPluginFixture(t *testing.T, cfg *config.Manager, root string, trust bool) *plugins.Manager {
	t.Helper()
	mgr, err := plugins.NewManager(filepath.Join(cfg.HomeDir(), "plugins"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Install(context.Background(), root, plugins.InstallOptions{Trust: trust}); err != nil {
		t.Fatal(err)
	}
	return mgr
}

func TestPluginSkillsRemainReadableAcrossLooseReload(t *testing.T) {
	cfg := pluginTestConfig(t)
	root := pluginFixture(t, map[string]string{
		"skills/review/SKILL.md": "---\nname: review\ndescription: Review changed code\n---\nInspect the requested code with evidence.",
		"commands/Review.md":     "---\ndescription: Review\n---\nInspect $ARGUMENTS",
	})
	installPluginFixture(t, cfg, root, false)
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	loose := t.TempDir()
	if err := os.MkdirAll(filepath.Join(loose, "local"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(loose, "local", "SKILL.md"), []byte("# local\n\nLocal workflow"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := a.LoadSkills(loose); err != nil {
			t.Fatal(err)
		}
		if len(a.Skills()) != 2 {
			t.Fatalf("loose reload lost plugin skills: %+v", a.Skills())
		}
		out, err := a.Tools().Call("skill_read", map[string]any{"name": "fixture:review", "detail": "full"})
		if err != nil || !strings.Contains(out, "Inspect the requested code") {
			t.Fatalf("skill_read=%q err=%v", out, err)
		}
	}
	if len(a.Tools().ListByCategory(pluginCategory)) != 1 {
		t.Fatal("untrusted instruction-only skill acquired executable tools")
	}
	input := MultimodalUserTurnInput("  /fixture:Review some-file.go ", []turn.Attachment{{Type: turn.AttachmentImage, FileName: "diagram.png"}}).WithScope(TurnScope{Platform: "telegram", SenderID: "a"})
	expanded := a.expandPluginCommand(input)
	if !strings.Contains(expanded.RoutingText, "Inspect some-file.go") || strings.Contains(expanded.RoutingText, "Inspect /fixture:") {
		t.Fatal(expanded.RoutingText)
	}
	if expanded.OriginalText != input.OriginalText || len(expanded.Attachments) != 1 || expanded.Scope.SenderID != "a" {
		t.Fatal("expansion lost original turn metadata")
	}
	if again := a.expandPluginCommand(expanded); again.RoutingText != expanded.RoutingText {
		t.Fatal("command expanded twice")
	}
}

func TestPluginCommandPreservesGatewayContext(t *testing.T) {
	a := &Agent{plugins: &pluginRuntime{commands: map[string]plugins.Prompt{
		"fixture:review": {Name: "review", Content: "Inspect $ARGUMENTS"},
	}}}
	input := MultimodalUserTurnInput("/fixture:review MixedCase.go --Keep-Case", []turn.Attachment{{Type: turn.AttachmentImage, FileName: "diagram.png"}}).
		WithScope(TurnScope{Platform: "telegram", ChatID: "room", SenderID: "user"})
	for _, routing := range []string{
		"[Replied message]\nReview this change\n\n" + input.OriginalText + "\n\n[Delivery guidance]\nReturn evidence",
		"[Replied message]\nReview this change\n\n[Delivery guidance]\nReturn evidence",
	} {
		expanded := a.expandPluginCommand(input.WithRoutingText(routing))
		if !strings.Contains(expanded.RoutingText, "Review this change") || !strings.Contains(expanded.RoutingText, "Return evidence") {
			t.Fatalf("command expansion discarded gateway context: %s", expanded.RoutingText)
		}
		if !strings.Contains(expanded.RoutingText, "Inspect MixedCase.go --Keep-Case") {
			t.Fatal("command arguments changed")
		}
		if expanded.OriginalText != input.OriginalText || expanded.Scope != input.Scope || len(expanded.Attachments) != 1 {
			t.Fatal("command expansion discarded original turn metadata")
		}
		if again := a.expandPluginCommand(expanded); again.RoutingText != expanded.RoutingText {
			t.Fatal("wrapped command expanded more than once")
		}
	}
}

func newPluginScriptTestAgent(t *testing.T, script string) (*Agent, string, *session.Session) {
	t.Helper()
	requirePluginScriptProcessGroups(t)
	requireShForHooks(t)
	cfg := pluginTestConfig(t)
	root := pluginFixture(t, map[string]string{"skills/check/SKILL.md": "# check\n\nCheck data.\n\n## Tools\n\n- `echo`: Echo\n", "skills/check/scripts/echo.sh": script})
	installPluginFixture(t, cfg, root, true)
	sessions, err := session.NewManager(filepath.Join(cfg.HomeDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{cfg: cfg, tools: tool.NewRegistry(), hitl: newHITLGate(), sessions: sessions}
	a.gateway = tool.NewGateway(a.tools)
	if err := a.preparePlugins(); err != nil {
		t.Fatal(err)
	}
	if err := a.registerPluginTools(); err != nil {
		t.Fatal(err)
	}
	return a, "plugin_fixture_check_echo", sessions.New()
}

func TestPluginScriptRequiresApprovalAndRejectsIso(t *testing.T) {
	a, name, sess := newPluginScriptTestAgent(t, "printf '%s' \"$SKILL_ARG_VALUE\"\n")
	if _, err := a.Tools().Call(name, nil); err == nil {
		t.Fatal("legacy entry executed without approval")
	}
	_, err := a.Tools().CallDetailedWithContext(name, nil, tool.ExecutionContext{Context: context.Background()})
	var approval *tool.ApprovalRequiredError
	if !errors.As(err, &approval) {
		t.Fatalf("missing approval: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := a.executeToolWithSessionDetailedContext(ctx, name, `{"value":"approved"}`, false, sess, "cli")
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for len(a.hitl.list()) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	pending := a.hitl.list()
	if len(pending) != 1 {
		t.Fatal("plugin call skipped the host HITL gate")
	}
	if _, err := a.hitl.Resolve(pending[0].ID, "deny", ""); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err == nil {
		t.Fatal("denied script ran")
	}
	out, err := a.Tools().CallDetailedWithContext(name, map[string]any{"value": "literal; $(not-shell)"}, tool.ExecutionContext{Context: context.Background(), AutoApprove: true})
	if err != nil || out.Output != "literal; $(not-shell)" {
		t.Fatalf("output=%q err=%v", out.Output, err)
	}
	iso := &sandbox.Snapshot{Mode: sandbox.ModeIso, Root: t.TempDir()}
	if _, err := a.Tools().CallDetailedWithContext(name, nil, tool.ExecutionContext{Context: context.Background(), AutoApprove: true, Sandbox: iso}); err == nil {
		t.Fatal("plugin script escaped iso")
	}
	cfg := DefaultLoopConfig()
	a.applySandboxToolPolicy(&cfg, iso)
	if !toolNameInSet(makeToolNameSet(cfg.DisabledTools), name) {
		t.Fatal("iso model catalog contains plugin script")
	}
}

func TestPluginScriptCancellationAndOutputBound(t *testing.T) {
	a, name, _ := newPluginScriptTestAgent(t, "sleep 2\nprintf done\n")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := a.Tools().CallDetailedWithContext(name, nil, tool.ExecutionContext{Context: ctx, AutoApprove: true})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("unbounded script: elapsed=%s err=%v", time.Since(start), err)
	}
	buffer := &pluginBoundedOutput{limit: 64}
	_, _ = buffer.Write([]byte(strings.Repeat("x", 1000)))
	if len(buffer.String()) > 150 || !strings.Contains(buffer.String(), "truncated") {
		t.Fatal("output limit failed")
	}
}

func TestPluginScriptPreservesParentTurnRestrictionsWithAutoApproval(t *testing.T) {
	a, name, sess := newPluginScriptTestAgent(t, "printf executed > \"$SKILL_ARG_MARKER\"\n")
	for _, tc := range []struct {
		name    string
		request string
		config  LoopConfig
		action  string
		reason  string
	}{
		{name: "read-only", request: "只读查看这个技能", reason: "without executing the skill"},
		{name: "instructions-only", request: "读取说明即可", reason: "without executing the skill"},
		{name: "no-execution", request: "不要执行技能", reason: "without executing the skill"},
		{name: "no-running", request: "不要运行这个技能", reason: "without executing the skill"},
		{name: "disabled-real-name", config: LoopConfig{DisabledTools: []string{name}}, reason: "disabled for this turn"},
		{name: "blocked-real-action", config: LoopConfig{BlockedActions: map[string][]string{name: {"write"}}}, action: "write", reason: "action is disabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "must-not-exist")
			arguments := fmt.Sprintf(`{"marker":%q,"action":%q}`, marker, tc.action)
			ctx := withPluginCallPolicy(context.Background(), tc.config, TurnScope{}, tc.request)
			_, err := a.executeToolWithSessionDetailedContext(ctx, name, arguments, true, sess, "cli")
			if err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("parent restriction was not enforced: %v", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("restricted plugin wrote its marker: %v", err)
			}
		})
	}
}

func TestPluginHooksMergeAndSurviveConfigReload(t *testing.T) {
	requireShForHooks(t)
	cfg := pluginTestConfig(t)
	root := pluginFixture(t, map[string]string{"hooks/hooks.json": `{"pre_tool_use":[{"match":["file_delete"],"command":"printf '%s' '{\"decision\":\"block\",\"reason\":\"plugin policy\"}'"}]}`})
	installPluginFixture(t, cfg, root, true)
	a := &Agent{cfg: cfg, hooks: hook.NewRunner(hook.Config{})}
	if err := a.preparePlugins(); err != nil {
		t.Fatal(err)
	}
	c := cfg.Get()
	c.Hooks.Enabled = false
	c.Hooks.PreToolUse = []config.HookSpec{{Match: []string{"file_read"}, Command: `echo '{"decision":"block","reason":"disabled user hook"}'`}}
	for i := 0; i < 2; i++ {
		a.ReloadHooks(c)
		_, blocked, reason := a.hooks.RunPre("file_delete", "{}", "http", "s")
		if !blocked || !strings.Contains(reason, "plugin policy") {
			t.Fatalf("hook did not block: %s", reason)
		}
		if _, blocked, _ := a.hooks.RunPre("file_read", "{}", "http", "s"); blocked {
			t.Fatal("plugin enabled disabled user hooks")
		}
	}
}

func TestPluginMCPStartupCallReconnectAndApproval(t *testing.T) {
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "plugin-test", Version: "1"}, nil)
	mcpServer.AddTool(&mcp.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "plugin-mcp-called"}}}, nil
	})
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return mcpServer }, &mcp.StreamableHTTPOptions{Stateless: true}))
	defer httpServer.Close()
	cfg := pluginTestConfig(t)
	root := pluginFixture(t, map[string]string{".mcp.json": fmt.Sprintf(`{"mcpServers":{"docs":{"type":"http","url":%q}}}`, httpServer.URL)})
	installPluginFixture(t, cfg, root, true)
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	name := tool.MCPToolName("fixture.docs", "echo")
	out, err := a.Tools().CallWithContext(name, nil, tool.ExecutionContext{Context: context.Background(), AutoApprove: true})
	if err != nil || out != "plugin-mcp-called" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if err := a.MCPAdmin().Disconnect(context.Background(), "fixture.docs"); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.Tools().Get(name); ok {
		t.Fatal("disconnect retained tool")
	}
	if err := a.MCPAdmin().ConnectConfigured(context.Background(), "fixture.docs"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, _, err = a.gateToolApproval(ctx, a.Sessions().New().ID, name, nil, false)
	if err == nil {
		t.Fatal("MCP PermApprove was not enforced")
	}
	isoCtx := sandbox.WithSnapshot(context.Background(), &sandbox.Snapshot{Mode: sandbox.ModeIso, Root: t.TempDir()})
	if _, err := a.executeToolWithSessionDetailedContext(isoCtx, name, "{}", true, nil, "cli"); err == nil {
		t.Fatal("MCP escaped iso execution policy")
	}
}

type pluginRecordingProvider struct {
	mu       sync.Mutex
	messages []provider.Message
	policy   pluginCallPolicy
	options  provider.CallOptions
}

func (p *pluginRecordingProvider) Name() string    { return "plugin-fixture" }
func (p *pluginRecordingProvider) Validate() error { return nil }
func (p *pluginRecordingProvider) Chat(ctx context.Context, messages []provider.Message) (*provider.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messages = append([]provider.Message(nil), messages...)
	p.policy, _ = pluginPolicyFromContext(ctx)
	return &provider.Response{Content: "reviewed"}, nil
}
func (p *pluginRecordingProvider) ChatStream(context.Context, []provider.Message) (<-chan provider.StreamChunk, error) {
	return nil, fmt.Errorf("unexpected stream")
}
func (p *pluginRecordingProvider) ChatWithOptions(ctx context.Context, messages []provider.Message, opts provider.CallOptions) (*provider.Response, error) {
	p.options = opts
	return p.Chat(ctx, messages)
}
func (p *pluginRecordingProvider) ChatStreamWithOptions(context.Context, []provider.Message, provider.CallOptions) (<-chan provider.StreamChunk, error) {
	return nil, fmt.Errorf("unexpected stream")
}

func TestPluginAgentInheritsParentPolicyAndScope(t *testing.T) {
	cfg := pluginTestConfig(t)
	root := pluginFixture(t, map[string]string{"agents/reviewer.md": "---\ndescription: reviewer\n---\nReview with evidence."})
	installPluginFixture(t, cfg, root, false)
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	recorder := &pluginRecordingProvider{}
	a.providerMu.Lock()
	a.provider = recorder
	a.providerMu.Unlock()
	parent := a.Sessions().New()
	parent.SetCwd(t.TempDir())
	parentCfg := DefaultLoopConfig()
	parentCfg.SessionID = parent.ID
	parentCfg.Source = "telegram"
	parentCfg.MaxIterations = 2
	parentCfg.Timeout = time.Second
	parentCfg.DisabledTools = []string{"terminal"}
	parentCfg.RestrictTools = true
	parentCfg.AllowedTools = []string{"file_read"}
	parentCfg.BlockedActions = map[string][]string{"file_write": {"write"}}
	scope := TurnScope{Platform: "telegram", ChatType: "private", SenderID: "user-a", ChatID: "room-a"}
	ctx := withPluginCallPolicy(context.Background(), parentCfg, scope, "Inspect only; do not modify files")
	exec := tool.ExecutionContext{Context: ctx, SessionID: parent.ID, Source: "telegram", AutoApprove: true}
	out, err := a.runPluginAgent(exec, "fixture:reviewer", "review source")
	if err != nil || out != "reviewed" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if recorder.policy.Scope != scope || recorder.policy.Config.Source != "telegram" || recorder.policy.Config.AutoApprove || recorder.policy.Config.MaxIterations != 2 {
		t.Fatalf("lost caller boundaries: %+v", recorder.policy)
	}
	if !toolNameInSet(makeToolNameSet(recorder.policy.Config.DisabledTools), "terminal") || !toolNameInSet(makeToolNameSet(recorder.policy.Config.DisabledTools), "plugin_agent") {
		t.Fatal("child regained tools")
	}
	if recorder.policy.Depth != 1 {
		t.Fatal("child lost recursion guard")
	}
	if !recorder.policy.Config.RestrictTools || len(recorder.policy.Config.AllowedTools) != 1 || recorder.policy.Config.AllowedTools[0] != "file_read" {
		t.Fatal("plugin child lost the parent role tool allowlist")
	}
	for _, schema := range recorder.options.Tools {
		if functionToolNameFromSchema(schema) != "file_read" {
			t.Fatalf("plugin child offered a tool outside its parent allowlist: %s", functionToolNameFromSchema(schema))
		}
	}
	joined := ""
	for _, m := range recorder.messages {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "Review with evidence") || !strings.Contains(joined, "do not modify files") {
		t.Fatalf("child lost preset=%t or parent constraints=%t: %s", !strings.Contains(joined, "Review with evidence"), !strings.Contains(joined, "do not modify files"), truncate(joined, 2500))
	}
	if _, err := a.runPluginAgent(tool.ExecutionContext{Context: pluginChildContext(ctx), SessionID: parent.ID, AutoApprove: true}, "fixture:reviewer", "again"); err == nil {
		t.Fatal("recursive plugin agent accepted")
	}
	if _, err := a.runPluginAgent(tool.ExecutionContext{Context: ctx, SessionID: "missing", AutoApprove: true}, "fixture:reviewer", "review"); err == nil {
		t.Fatal("missing parent accepted")
	}
}

func TestPluginPromptExecutionOverridesAreRejected(t *testing.T) {
	for _, key := range []string{"model", "tools", "permission_mode", "maxTurns", "isolation", "background"} {
		if err := validatePluginPrompts(&plugins.Bundle{Manifest: plugins.Manifest{Name: "test"}, Agents: []plugins.Prompt{{Name: "review", Content: "---\n" + key + ": true\n---\nReview"}}}); err == nil {
			t.Errorf("silently accepted %s", key)
		}
	}
}
