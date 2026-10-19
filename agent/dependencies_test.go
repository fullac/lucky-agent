package agent

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurika0211/luckyagent-runtime/bootstrap"
	"github.com/yurika0211/luckyagent-runtime/config"
	"github.com/yurika0211/luckyagent-runtime/mcpclient"
	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-runtime/session"
	"github.com/yurika0211/luckyagent-runtime/tool"
	"github.com/yurika0211/luckyagent-tools/memory"
)

func TestInjectedMinimalToolLoopNeedsNoMemoryOrSchedulers(t *testing.T) {
	root := t.TempDir()
	cfg, err := config.NewManagerWithDir(root)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := session.NewManager(filepath.Join(root, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	registry := tool.NewRegistry()
	calls := 0
	registry.Register(&tool.Tool{Name: "fixture_echo", Permission: tool.PermAuto, Enabled: true, Handler: func(map[string]any) (string, error) { calls++; return "ECHO_CONFIRMED", nil }})
	p := &requestCaptureProvider{chat: func(messages []provider.Message, opts provider.CallOptions) (*provider.Response, error) {
		if len(opts.Tools) == 0 {
			return &provider.Response{Content: verifiedAssessment}, nil
		}
		for _, m := range messages {
			if m.Role == "tool" && m.Content == "ECHO_CONFIRMED" {
				return &provider.Response{Content: "ECHO_CONFIRMED"}, nil
			}
		}
		return &provider.Response{ToolCalls: []provider.ToolCall{{ID: "echo-1", Name: "fixture_echo", Arguments: "{}"}}}, nil
	}}
	deps := &Dependencies{Config: cfg, Sessions: sessions, Provider: bootstrap.ProviderRuntime{Provider: p}, Support: bootstrap.SupportRuntime{Tools: registry}}
	before := treeNames(t, root)
	a, err := NewWithDependencies(Options{TakeOwnership: true}, deps)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if after := treeNames(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("minimal binding created files")
	}
	if a.memory != nil || a.delegate != nil || a.autonomy != nil || a.toolServices != nil {
		t.Fatal("constructor created optional backends")
	}
	for _, name := range []string{"subagent", "autonomy", "cron"} {
		if _, ok := registry.Get(name); ok {
			t.Fatalf("unavailable capability %s registered", name)
		}
	}
	result, err := a.Chat(context.Background(), "Please call fixture_echo once and report its result")
	if err != nil || result != "ECHO_CONFIRMED" || calls != 1 {
		t.Fatalf("minimal tool loop failed: response=%q calls=%d err=%v", result, calls, err)
	}
}

func preparedTestComponents(t *testing.T, endpoint string) *bootstrap.Components {
	t.Helper()
	mgr, err := config.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := mgr.Get()
	cfg.APIKey, cfg.LlmProvider.APIKey = "test-key", "test-key"
	cfg.Proactive.Enabled = true
	if endpoint != "" {
		cfg.MCP.Servers = map[string]mcpclient.ServerConfig{"fixture": {Transport: "http", URL: endpoint, TimeoutSeconds: 1}}
	}
	if err := mgr.Replace(cfg); err != nil {
		t.Fatal(err)
	}
	components, err := bootstrap.Open(mgr, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = components.Close() })
	return components
}

func treeNames(t *testing.T, directory string) []string {
	t.Helper()
	var result []string
	err := filepath.WalkDir(directory, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		result = append(result, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestInjectedConstructionDoesNotActivateServicesOrCreateFiles(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	components := preparedTestComponents(t, server.URL)
	before := treeNames(t, components.Config.HomeDir())
	instance, err := NewWithDependencies(Options{ContextCacheSize: 4}, components)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	_ = instance.StartupHooks()
	if requests.Load() != 0 || components.ProactiveRuntime.Started() || components.Support.AutonomyKit.Status().Started {
		t.Fatal("injected construction activated an application service")
	}
	if after := treeNames(t, components.Config.HomeDir()); !reflect.DeepEqual(before, after) {
		t.Fatalf("constructor created files: before=%v after=%v", before, after)
	}
	if instance.Sessions() != components.Sessions || instance.Provider() != components.Provider.Provider {
		t.Fatal("injected resources were replaced")
	}
}

type countingMemoryStore struct {
	memory.RuntimeStore
	closed int
}

func (s *countingMemoryStore) Close() error {
	s.closed++
	return s.RuntimeStore.Close()
}

func TestInjectedResourceOwnershipAndRepeatedClose(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(map[bool]string{false: "borrowed", true: "owned"}[owned], func(t *testing.T) {
			components := preparedTestComponents(t, "")
			store := &countingMemoryStore{RuntimeStore: components.Memory.Store}
			components.Memory.Store = store
			instance, err := NewWithDependencies(Options{TakeOwnership: owned}, components)
			if err != nil {
				t.Fatal(err)
			}
			if err := instance.Close(); err != nil {
				t.Fatal(err)
			}
			if err := instance.Close(); err != nil {
				t.Fatal(err)
			}
			want := 0
			if owned {
				want = 1
			}
			if store.closed != want {
				t.Fatalf("close calls=%d want=%d", store.closed, want)
			}
			if err := components.Close(); err != nil || store.closed != 1 {
				t.Fatalf("owner close=%d err=%v", store.closed, err)
			}
		})
	}
}

func TestInvalidInjectionDoesNotTakeOwnership(t *testing.T) {
	components := preparedTestComponents(t, "")
	store := &countingMemoryStore{RuntimeStore: components.Memory.Store}
	components.Memory.Store = store
	components.Provider.Provider = nil
	if _, err := NewWithDependencies(Options{TakeOwnership: true}, components); err == nil || store.closed != 0 {
		t.Fatal("invalid constructor accepted dependencies or closed caller resources")
	}
}

func TestInjectedPluginSnapshotWaitsForExplicitStartup(t *testing.T) {
	var requests atomic.Int32
	backend := mcp.NewServer(&mcp.Implementation{Name: "injected-plugin", Version: "1"}, nil)
	backend.AddTool(&mcp.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "injected-plugin-called"}}}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return backend }, &mcp.StreamableHTTPOptions{Stateless: true})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	components := preparedTestComponents(t, "")
	root := pluginFixture(t, map[string]string{
		"commands/review.md": "Review $ARGUMENTS",
		".mcp.json":          fmt.Sprintf(`{"mcpServers":{"docs":{"type":"http","url":%q}}}`, server.URL),
	})
	installPluginFixture(t, components.Config, root, true)
	before := treeNames(t, components.Config.HomeDir())
	instance, err := NewWithDependencies(Options{}, components)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	if !instance.HasPluginCommand("/fixture:review source.go") {
		t.Fatal("injected construction omitted the plugin snapshot")
	}
	if requests.Load() != 0 {
		t.Fatal("injected construction started plugin MCP")
	}
	if after := treeNames(t, components.Config.HomeDir()); !reflect.DeepEqual(before, after) {
		t.Fatal("injected plugin construction created runtime files")
	}
	if err := components.Start(context.Background(), instance.StartupHooks()); err != nil {
		t.Fatal(err)
	}
	name := tool.MCPToolName("fixture.docs", "echo")
	output, err := instance.Tools().CallWithContext(name, nil, tool.ExecutionContext{Context: context.Background(), AutoApprove: true})
	if err != nil || output != "injected-plugin-called" || requests.Load() == 0 {
		t.Fatalf("explicit startup did not activate plugin MCP: output=%q err=%v", output, err)
	}
}

func TestInjectedPluginsPreservePreparedMCPSettings(t *testing.T) {
	components := preparedTestComponents(t, "")
	host := mcpclient.Config{Servers: map[string]mcpclient.ServerConfig{
		"custom":       {URL: "http://127.0.0.1:1", Disabled: true},
		"fixture.docs": {URL: "http://127.0.0.1:2", Disabled: true},
	}}
	if err := components.MCP.Configure(host); err != nil {
		t.Fatal(err)
	}
	root := pluginFixture(t, map[string]string{
		".mcp.json": `{"mcpServers":{"docs":{"type":"http","url":"http://127.0.0.1:3"},"extra":{"type":"http","url":"http://127.0.0.1:4"}}}`,
	})
	installPluginFixture(t, components.Config, root, true)
	instance, err := NewWithDependencies(Options{}, components)
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	actual := components.MCP.Configured()
	if len(actual.Servers) != 3 || actual.Servers["fixture.extra"].URL != "http://127.0.0.1:4" {
		t.Fatalf("plugin MCP declarations missing: %+v", actual)
	}
	for name, expected := range host.Servers {
		if !reflect.DeepEqual(actual.Servers[name], expected) {
			t.Fatalf("plugin replaced prepared server %s: %+v", name, actual.Servers[name])
		}
	}
	if len(components.MCP.Status()) != 0 {
		t.Fatal("injected configuration activated MCP")
	}
}
