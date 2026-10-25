package agent

import (
	"strings"

	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
	compactlib "github.com/yurika0211/luckyagent-tools/compact"
)

func generateLocalCompactSummary(messages []provider.Message, _ *contextx.TokenEstimator) string {
	summary := compactlib.LocalSummary(toCompactMessages(messages), compactTextOptions())
	identifiers := requiredCompactIdentifiers(messages)
	if len(missingCompactIdentifiers(summary, identifiers)) == 0 {
		return summary
	}
	// LocalSummary limits its file/command lists. Preserve the required set in
	// an extractive appendix instead of weakening the validator for local mode.
	appendix := "\n\nSource identifiers (verbatim tool evidence; consult the original transcript):\n- " + strings.Join(identifiers, "\n- ")
	room := compactlib.MaxSummaryRunes - len([]rune(appendix))
	runes := []rune(summary)
	if len(runes) > room {
		marker := "\n[Local summary details omitted; source identifiers follow.]"
		runes = runes[:room-len([]rune(marker))]
		summary = string(runes) + marker
	}
	return summary + appendix
}
