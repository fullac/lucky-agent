package agent

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/yurika0211/luckyagent-runtime/provider"
)

func TestCheckpointLegacyResultMigrationPreservesEquality(t *testing.T) {
	result := strings.Repeat("tool-result", 1000)
	cp := durableCheckpoint{Version: 1, LastResults: map[string]string{"signature": result}, Messages: []provider.Message{{Role: "tool", Content: result}}}
	before, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	cp.migrateResultDigests()
	after, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	var restored durableCheckpoint
	if err := json.Unmarshal(after, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.LastResultDigests["signature"] != durableResultDigest(result) || restored.Messages[0].Content != result || restored.LastResults != nil {
		t.Fatal("checkpoint migration lost equality state or full evidence")
	}
	if len(after) >= len(before) {
		t.Fatal("migration did not remove redundant output copy")
	}
}

// Measures the cumulative cost of saving after every operation, rather than
// extrapolating from a single small checkpoint. Run with -benchtime=1x.
func BenchmarkCheckpointTrajectory(b *testing.B) {
	for _, steps := range []int{100, 1000} {
		for _, digests := range []bool{false, true} {
			name := fmt.Sprintf("operations=%d/digests=%t", steps, digests)
			b.Run(name, func(b *testing.B) {
				body := strings.Repeat("output-", 146)
				var totalBytes int64
				var durations []int64
				b.ReportAllocs()
				b.ResetTimer()
				for n := 0; n < b.N; n++ {
					cp := durableCheckpoint{Version: 1, LastResults: map[string]string{}, LastResultDigests: map[string]string{}}
					for i := 0; i < steps; i++ {
						id := fmt.Sprintf("operation-%d", i)
						cp.Messages = append(cp.Messages, provider.Message{Role: "tool", ToolCallID: id, Content: body})
						if digests {
							cp.LastResultDigests[id] = durableResultDigest(body)
						} else {
							cp.LastResults[id] = body
						}
						start := time.Now()
						data, err := json.Marshal(cp)
						if err != nil {
							b.Fatal(err)
						}
						durations = append(durations, time.Since(start).Nanoseconds())
						totalBytes += int64(len(data))
					}
				}
				b.StopTimer()
				sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
				b.ReportMetric(float64(totalBytes)/float64(b.N), "encoded-bytes/trajectory")
				b.ReportMetric(float64(durations[(len(durations)-1)*95/100]), "encode-p95-ns")
			})
		}
	}
}
