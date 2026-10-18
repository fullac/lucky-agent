package agent

import (
	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
	compactlib "github.com/yurika0211/luckyagent-tools/compact"
)

func generateLocalCompactSummary(messages []provider.Message, _ *contextx.TokenEstimator) string {
	return compactlib.LocalSummary(toCompactMessages(messages), compactTextOptions())
}
