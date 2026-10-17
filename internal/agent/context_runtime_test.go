package agent

import (
	"context"
	"testing"

	"github.com/yurika0211/luckyagent/internal/config"
	"github.com/yurika0211/luckyagent/internal/contextx"
	"github.com/yurika0211/luckyagent/internal/memory"
	"github.com/yurika0211/luckyagent/internal/provider"
	"github.com/yurika0211/luckyagent/internal/rag"
	"github.com/yurika0211/luckyagent/internal/session"
	"github.com/yurika0211/luckyagent/internal/soul"
	"github.com/yurika0211/luckyagent/internal/tool"
	"github.com/yurika0211/luckyagent/internal/turn"
)

// contextRuntimeStub exercises the planner without constructing an Agent.
// This keeps the runtime contract honest as the coordinator grows.
type contextRuntimeStub struct{}

func (contextRuntimeStub) runContextMemoryHygieneHook() {}

func (contextRuntimeStub) buildSystemPromptWithOptions(*session.Session, systemPromptOptions) string {
	return "stub system prompt"
}

func (contextRuntimeStub) buildSkillRouteSystemHintWithOptions(string, skillRouteOptions) string {
	return ""
}

func (contextRuntimeStub) fitContextWindow(messages []provider.Message) []provider.Message {
	return messages
}

func (contextRuntimeStub) AnalyzeAttachments(context.Context, []turn.Attachment) (string, error) {
	return "", nil
}

func (contextRuntimeStub) Config() *config.Manager                    { return nil }
func (contextRuntimeStub) ContextWindow() *contextx.ContextWindow     { return nil }
func (contextRuntimeStub) ContextEstimator() *contextx.TokenEstimator { return nil }
func (contextRuntimeStub) ContextCache() contextMessageCacheStore     { return nil }
func (contextRuntimeStub) Soul() *soul.Soul                           { return nil }
func (contextRuntimeStub) ContextMemory() memory.ContextReader        { return nil }
func (contextRuntimeStub) MidTermStore() memory.MidTermStore          { return nil }
func (contextRuntimeStub) ShortTermSummary(*session.Session) string   { return "" }
func (contextRuntimeStub) Catalog() *provider.ModelCatalog            { return nil }
func (contextRuntimeStub) Tools() *tool.Registry                      { return nil }
func (contextRuntimeStub) RAG() *rag.RAGManager                       { return nil }
func (contextRuntimeStub) MemoryVaultPath() string                    { return "" }

func TestContextPlannerAcceptsRuntimeDouble(t *testing.T) {
	planner := &contextPlanner{
		agent: contextRuntimeStub{},
		est:   contextx.NewTokenEstimator(4096),
		budget: contextBudget{
			System: 256, Memory: 128, RAG: 256, History: 256, ToolResult: 256,
		},
		options: contextBuildOptions{IncludeHistory: false, IncludeRAG: false},
	}

	messages := planner.Build(context.Background(), nil, "hello")
	if len(messages) != 2 {
		t.Fatalf("expected system and user messages, got %d: %#v", len(messages), messages)
	}
	if messages[0].Role != "system" || messages[0].Content != "stub system prompt" {
		t.Fatalf("unexpected system message: %#v", messages[0])
	}
	if messages[1].Role != "user" || messages[1].Content != "hello" {
		t.Fatalf("unexpected user message: %#v", messages[1])
	}
}
