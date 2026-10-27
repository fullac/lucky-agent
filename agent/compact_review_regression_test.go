package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
)

func TestLocalCompactRetainsAllRequiredSourceIdentifiers(t *testing.T) {
	messages := []provider.Message{{Role: "user", Content: "Investigate these failures and keep the paths."}}
	for i := 0; i < 20; i++ {
		messages = append(messages, provider.Message{Role: "tool", Name: "terminal",
			Content: fmt.Sprintf("error: operation %d failed at /srv/CaseSensitive/module%d.go", i, i)})
	}
	summary := generateLocalCompactSummary(messages, nil)
	if validation := validateCompactSummary(summary, messages); !validation.Valid {
		t.Fatalf("deterministic fallback cannot pass its own gate: %s\n%s", validation.Reason, summary)
	}
}

func TestCompactValidatorRequiresCommandsWithoutPaths(t *testing.T) {
	messages := []provider.Message{{Role: "tool", Name: "terminal", Content: "git status --short"}}
	if got := requiredCompactIdentifiers(messages); len(got) != 1 || got[0] != "git status --short" {
		t.Fatalf("command without a path was not captured: %v", got)
	}
}

func TestCompactIdentifierMatchingPreservesCase(t *testing.T) {
	if got := missingCompactIdentifiers("read /srv/config.json", []string{"/srv/Config.json"}); len(got) != 1 {
		t.Fatal("different case-sensitive filesystem path was accepted")
	}
}

func TestCompactFingerprintIncludesStructuredEvidence(t *testing.T) {
	original := []provider.Message{{Role: "assistant", Content: "checking", ToolCalls: []provider.ToolCall{
		{ID: "call", Name: "file_read", Arguments: `{"path":"first.txt"}`},
	}}}
	changed := []provider.Message{{Role: "assistant", Content: "checking", ToolCalls: []provider.ToolCall{
		{ID: "call", Name: "file_read", Arguments: `{"path":"second.txt"}`},
	}}}
	if compactRangeFingerprint(original, 0, 1) == compactRangeFingerprint(changed, 0, 1) {
		t.Fatal("tool arguments changed without invalidating the compaction snapshot")
	}
}

func BenchmarkCompactRepeatedLines(b *testing.B) {
	text := strings.Repeat("ordinary repeated line\n", 10000) + "error: /srv/app/main.go failed"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		got := compressToolResultLines(text, 400)
		if len([]rune(got)) > 400 || !strings.Contains(got, "/srv/app/main.go") {
			b.Fatal("compression lost the error or exceeded the budget")
		}
	}
}

func TestToolEvidenceFitsWithoutMonotonicLineAssumption(t *testing.T) {
	p := &contextPlanner{est: contextx.NewTokenEstimator(4096)}
	text := strings.Repeat("普通说明 text repeated\n", 100) + "error: /srv/CaseSensitive/main.go failed"
	for _, budget := range []int{1, 4, 16, 64, 128, 256} {
		got := p.fitToolEvidenceToBudget(text, budget)
		if got == "" || p.est.Estimate(got) > budget {
			t.Fatalf("budget %d returned unusable evidence (%d tokens): %q", budget, p.est.Estimate(got), got)
		}
	}
}
