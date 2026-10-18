package agent

import (
	"github.com/yurika0211/luckyagent-runtime/provider"
	compactlib "github.com/yurika0211/luckyagent-tools/compact"
)

type CompactSummaryValidation = compactlib.Validation

func validateCompactSummary(summary string, messages []provider.Message) CompactSummaryValidation {
	return compactlib.Validate(summary, toCompactMessages(messages))
}
