package agent

import (
	"context"
	"fmt"

	"github.com/yurika0211/luckyagent/internal/config"
	"github.com/yurika0211/luckyagent/internal/memory"
	"github.com/yurika0211/luckyagent/internal/memoryaestus"
)

// Default backend selection is confined to this assembly point. The Agent
// loop and context planner operate only on the supplied memory ports.
func resolveMemoryRuntime(cfg *config.Manager, c *config.Config, injected *memory.Runtime) (memory.Runtime, error) {
	if injected != nil {
		if injected.Store == nil || injected.ShortTerm == nil || injected.Summaries == nil {
			return memory.Runtime{}, fmt.Errorf("incomplete memory runtime")
		}
		return *injected, nil
	}
	t := c.Memory.Tidal
	return memoryaestus.Open(memoryaestus.Options{HomeDir: cfg.HomeDir(), ShortTermMaxTurns: c.Memory.ShortTermMaxTurns,
		MidTermMaxSummaries: c.Memory.MidTermMaxSummaries, Tidal: memoryaestus.TidalOptions{
			Enabled: t.Enabled, StorePath: t.StorePath, Beta: t.Beta, MaxBoost: t.MaxBoost, LearningRate: t.LearningRate, MinSamples: t.MinSamples,
		}})
}

func (a *Agent) RecallMemory(ctx context.Context, request memory.RecallRequest) ([]memory.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a == nil || a.memory == nil {
		return nil, fmt.Errorf("memory runtime is unavailable")
	}
	return a.memory.Recall(ctx, request)
}
