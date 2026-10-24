package agent

import (
	"strings"
	"testing"

	"github.com/yurika0211/luckyagent-runtime/provider"
)

// These tests are the acceptance checks listed in #337. Each one asserts a
// property of the compaction rule instead of a specific output string.

// criterion: identifiers are never dropped. A path, an error head, and a value
// each appear on exactly one middle line and must survive in order.
func TestCompressToolResultKeepsIdentifiers(t *testing.T) {
	lines := []string{"Connecting to the build host"}
	for i := 0; i < 40; i++ {
		lines = append(lines, "progress step without payload")
	}
	lines = append(lines,
		"config written to /home/dev/workspace/lucky-agent/config.example.json",
		"another uninteresting status line",
		"error: dial tcp 10.0.0.7:8443: connection refused",
		"still nothing worth keeping",
		"elapsed=4213ms",
	)
	for i := 0; i < 40; i++ {
		lines = append(lines, "trailing chatter")
	}
	output := strings.Join(lines, "\n")

	compacted := compressToolResultLines(output, 400)
	if len([]rune(compacted)) > 400 {
		t.Fatalf("compacted output exceeds budget: %d runes", len([]rune(compacted)))
	}
	for _, want := range []string{
		"/home/dev/workspace/lucky-agent/config.example.json",
		"error: dial tcp 10.0.0.7:8443: connection refused",
		"elapsed=4213ms",
	} {
		if !strings.Contains(compacted, want) {
			t.Fatalf("identifier line dropped: %q\n%s", want, compacted)
		}
	}
	if strings.Index(compacted, "config.example.json") > strings.Index(compacted, "elapsed=4213ms") {
		t.Fatal("identifier lines lost their original order")
	}
}

// criterion: redundant lines are dropped before content. The budget only fits a
// few lines, so repeated headers, blank lines, and progress lines must go first.
func TestCompressToolResultDropsRedundantFirst(t *testing.T) {
	lines := []string{"name | status | note"}
	for i := 0; i < 20; i++ {
		lines = append(lines, "name | status | note", "", "loading...", "50%")
	}
	lines = append(lines, "alpha finished in 12.5s", "beta finished in 3.25s")
	output := strings.Join(lines, "\n")

	compacted := compressToolResultLines(output, 200)
	if len([]rune(compacted)) > 200 {
		t.Fatalf("compacted output exceeds budget: %d runes", len([]rune(compacted)))
	}
	for _, want := range []string{"alpha finished in 12.5s", "beta finished in 3.25s"} {
		if !strings.Contains(compacted, want) {
			t.Fatalf("informative line dropped before redundant ones: %q", want)
		}
	}
}

// criterion: a result never disappears. Even when nothing is protected and the
// budget is tiny, one original line and the omission record survive.
func TestCompressToolResultNeverDropsEveryLine(t *testing.T) {
	output := strings.Join([]string{"first scratch note", "second scratch note", "third scratch note"}, "\n")
	compacted := compressToolResultLines(output, 8)
	if strings.TrimSpace(compacted) == "" {
		t.Fatal("compaction removed the whole result")
	}
	if !strings.Contains(compacted, "omitted for context") && len([]rune(output)) > 8 {
		t.Fatal("expected an omission record pointing at the transcript")
	}
}

// criterion: retained lines are a subsequence of the original lines.
func TestCompressToolResultKeepsOrder(t *testing.T) {
	lines := make([]string, 0, 60)
	for i := 0; i < 20; i++ {
		lines = append(lines, "filler line", "", "go test ./internal/agent", "note")
	}
	output := strings.Join(lines, "\n")
	compacted := compressToolResultLines(output, 120)

	position := 0
	for _, line := range strings.Split(compacted, "\n") {
		if strings.Contains(line, "omitted for context") || strings.Contains(line, "middle omitted") {
			continue
		}
		found := false
		for ; position < len(lines); position++ {
			if lines[position] == line {
				found = true
				position++
				break
			}
		}
		if !found {
			t.Fatalf("retained line is not a subsequence: %q", line)
		}
	}
}

// criterion: behaviour inside the budget is unchanged.
func TestCompressToolResultUnchangedWithinBudget(t *testing.T) {
	output := "one\ntwo\nthree"
	if got := compressToolResultLines(output, 1000); got != output {
		t.Fatalf("in-budget result changed: %q", got)
	}
}

// compactSampleMessages is one covered range with three identifiers: a path, an
// error head, and a value.
func compactSampleMessages() []provider.Message {
	return []provider.Message{
		{Role: "user", Content: "修复构建失败。"},
		{Role: "assistant", Content: "先跑测试再改。"},
		{Role: "tool", Name: "terminal", ToolCallID: "call-1", Content: "go test ./internal/agent\nconfig written to /home/dev/workspace/lucky-agent/agent/compact_adapter.go\nerror: undefined: compactlib.Validate\nelapsed=4213ms"},
	}
}

const compactSampleSummaryBody = "当前用户目标：修复构建失败，并确认 compact 不再丢掉源消息里的标识符。已确认改动落在 /home/dev/workspace/lucky-agent/agent/compact_adapter.go，命令 go test ./internal/agent 报错 error: undefined: compactlib.Validate，耗时 elapsed=4213ms。"

// criterion: a summary that passes the shape checks but drops a source identifier
// is rejected, and the failure names the identifier.
func TestValidateCompactSummaryRejectsDroppedIdentifier(t *testing.T) {
	messages := compactSampleMessages()

	positive := compactSampleSummaryBody + "\n\npending work: rerun the suite."
	if got := validateCompactSummary(positive, messages); !got.Valid {
		t.Fatalf("positive sample rejected: %s", got.Reason)
	}

	negative := strings.ReplaceAll(positive, "elapsed=4213ms", "elapsed measured")
	got := validateCompactSummary(negative, messages)
	if got.Valid {
		t.Fatal("negative sample accepted after dropping a source identifier")
	}
	if !strings.Contains(got.Reason, "4213") {
		t.Fatalf("failure does not name the dropped identifier: %q", got.Reason)
	}
}

// criterion: verifier strength is measured, not assumed. Three negatives drop one
// identifier each; three positives only reword. sigma is reported below.
func TestCompactValidatorStrength(t *testing.T) {
	messages := compactSampleMessages()
	base := compactSampleSummaryBody + "\n\npending work: rerun the suite."

	negatives := []string{
		strings.ReplaceAll(base, "4213", "measured"),
		strings.ReplaceAll(base, "/home/dev/workspace/lucky-agent/agent/compact_adapter.go", "the adapter file"),
		strings.ReplaceAll(base, "compactlib.Validate", "the validator"),
	}
	positives := []string{
		base,
		strings.ReplaceAll(base, "已确认改动落在", "变更位置为"),
		strings.ReplaceAll(base, "修复构建失败", "修好构建异常"),
	}

	agree := 0
	total := len(negatives) + len(positives)
	for _, sample := range negatives {
		if !validateCompactSummary(sample, messages).Valid {
			agree++
		}
	}
	for _, sample := range positives {
		if validateCompactSummary(sample, messages).Valid {
			agree++
		}
	}
	sigma := float64(agree) / float64(total)
	t.Logf("validator strength sigma=%.2f over %d samples (%d negatives, %d positives)", sigma, total, len(negatives), len(positives))
	if sigma < 1.0 {
		t.Fatalf("validator strength regressed: sigma=%.2f", sigma)
	}
}
