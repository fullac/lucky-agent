package agent

import (
	"context"
	"fmt"
	"github.com/yurika0211/luckyagent-tools/memory"
)

func (a *Agent) RecallMemory(ctx context.Context, request memory.RecallRequest) ([]memory.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a == nil || a.memory == nil {
		return nil, fmt.Errorf("memory runtime is unavailable")
	}
	return a.memory.Recall(ctx, request)
}
