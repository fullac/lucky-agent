package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yurika0211/luckyagent-runtime/config"
	"github.com/yurika0211/luckyagent-runtime/credentials"
	"github.com/yurika0211/luckyagent-runtime/mcpclient"
	"github.com/yurika0211/luckyagent-runtime/tool"
)

func TestAgentMCPConfiguredStartup(t *testing.T) {
	const token = "mcp-fixture-secret"
	server := mcp.NewServer(&mcp.Implementation{Name: "startup-test", Version: "1"}, nil)
	server.AddTool(&mcp.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "connected"}}}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer h.Close()
	mgr, _ := config.NewManagerWithDir(t.TempDir())
	store, err := credentials.NewStore(mgr.HomeDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "mcp-test", "mcp", "default", token); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	c := mgr.Get()
	c.Provider = "openai"
	c.APIKey = "sk-test"
	c.Model = "gpt-3.5-turbo"
	c.MCP = mcpclient.Config{Servers: map[string]mcpclient.ServerConfig{"remote": {URL: h.URL, CredentialRef: "mcp-test"}}}
	mgr.SetConfig(c)
	a, err := New(mgr)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if statuses := a.MCPAdmin().Status(); len(statuses) != 1 || statuses[0].State != "connected" {
		t.Fatalf("configured startup failed: %v", statuses)
	}
	name := tool.MCPToolName("remote", "echo")
	if permission, err := a.Tools().CheckPermission(name); err != nil || permission != tool.PermApprove {
		t.Fatalf("MCP approval policy lost: %v %v", permission, err)
	}
	if out, err := a.Tools().CallWithContext(name, nil, tool.ExecutionContext{Context: context.Background()}); err != nil || out != "connected" {
		t.Fatalf("configured tool not callable: %q %v", out, err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.Tools().Get(name); ok {
		t.Fatal("runtime close retained MCP tool")
	}
}
