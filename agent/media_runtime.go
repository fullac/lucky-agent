package agent

import (
	"github.com/yurika0211/luckyagent-runtime/config"
)

func (a *Agent) reloadMediaRuntime(c *config.Config) {
	if a == nil || c == nil {
		return
	}
	next := buildMediaRuntime(c)
	a.mediaMu.Lock()
	a.mediaProcessor = next.Processor
	a.mediaMu.Unlock()
	if a.toolServices != nil {
		a.toolServices.ReloadMediaTools(a.tools, "", next.Processor, next.ImageGenerator, next.ImageDefaults, next.SpeechSynthesizer, next.TtsDefaults)
	}
}
