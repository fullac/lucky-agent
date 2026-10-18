package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yurika0211/luckyagent-runtime/config"
	"github.com/yurika0211/luckyagent-runtime/memoryaestus"
	"github.com/yurika0211/luckyagent-tools/memory"
)

func TestAutoSummarizeKeepsScopedFactsOutOfSessionSummaries(t *testing.T) {
	runtime, err := memoryaestus.Open(memoryaestus.Options{HomeDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Store.Close()
	a := &Agent{memory: runtime.Store, midTerm: runtime.Summaries, shortTerms: runtime.ShortTerm}
	expiry := time.Now().Add(time.Hour)
	scopes := []TurnScope{{Platform: "telegram", SenderID: "alice", ChatType: "private"}, {Platform: "telegram", SenderID: "bob", ChatType: "private"}}
	for _, scope := range scopes {
		for i := 0; i < 7; i++ {
			_, err := a.WriteMemory(t.Context(), memory.RememberRequest{Content: fmt.Sprintf("%s project fact number %d", scope.SenderID, i), Category: "project", Tier: memory.TierShort, Importance: .7,
				Options: memory.SaveOptions{Tags: []string{"scope:personal", scope.UserTag()}, ExpiresAt: &expiry}})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	a.autoSummarize()
	if a.midTerm.Count() != 0 {
		t.Fatal("durable facts copied into globally queried session summaries")
	}
	entries := fixtureMemory(a).ByTier(memory.TierMedium)
	if len(entries) != 2 {
		t.Fatalf("summaries=%d", len(entries))
	}
	for _, entry := range entries {
		if entry.ExpiresAt == nil || !entry.ExpiresAt.Equal(expiry) {
			t.Fatal("expiry lost")
		}
		visible := 0
		for _, scope := range scopes {
			if memoryEntryVisibleInScope(entry, scope) {
				visible++
				other := "alice"
				if scope.SenderID == "alice" {
					other = "bob"
				}
				if strings.Contains(entry.Content, other) {
					t.Fatal("cross-scope content in summary")
				}
			}
		}
		if visible != 1 {
			t.Fatalf("summary visible to %d scopes", visible)
		}
	}
}

func TestNewWithMemoryKeepsSessionHistorySeparate(t *testing.T) {
	memoryHome := t.TempDir()
	runtime, err := memoryaestus.Open(memoryaestus.Options{HomeDir: memoryHome})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.NewManagerWithDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Set("api_key", "test-key"); err != nil {
		t.Fatal(err)
	}
	a, err := NewWithMemory(cfg, runtime)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Memory().Source().Location != filepath.Join(memoryHome, "memory") {
		t.Fatal("injected backend ignored")
	}
	if a.Sessions() == nil {
		t.Fatal("application session manager missing")
	}
	if _, err := os.Stat(filepath.Join(memoryHome, "sessions")); !os.IsNotExist(err) {
		t.Fatal("backend constructed application session history")
	}
}

func TestShortTermSessionlessCallsHaveNoSharedFallback(t *testing.T) {
	a := &Agent{shortTerms: memoryaestus.NewSessionShortTermStore(2)}
	if a.shortTermBufferForSession(nil, true) != nil {
		t.Fatal("sessionless shared buffer created")
	}
	if a.ShortTermSummary(nil) != "" || a.shortTerms.SessionCount() != 0 {
		t.Fatal("sessionless state escaped isolation")
	}
}
