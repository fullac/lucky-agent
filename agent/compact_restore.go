package agent

import (
	"github.com/yurika0211/luckyagent-runtime/contextx"
	"github.com/yurika0211/luckyagent-runtime/provider"
	"github.com/yurika0211/luckyagent-runtime/session"
	compactlib "github.com/yurika0211/luckyagent-tools/compact"
)

func buildPostCompactAttachments(sess *session.Session, messages []provider.Message, est *contextx.TokenEstimator) []session.CompactAttachment {
	var shell compactlib.ShellState
	if sess != nil {
		shell = sess
	}
	return compactlib.RestoreAttachments(shell, toCompactMessages(messages), compactEstimator(est), compactTextOptions())
}
