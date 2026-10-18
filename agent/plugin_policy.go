package agent

import "context"

type pluginPolicyKey struct{}

// This value belongs to one turn, not the Agent. Plugin children must inherit
// the caller's scope and restrictions rather than reconstructing global defaults.
type pluginCallPolicy struct {
	Config      LoopConfig
	Scope       TurnScope
	UserRequest string
	Depth       int
}

func withPluginCallPolicy(ctx context.Context, cfg LoopConfig, scope TurnScope, request string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	previous, _ := pluginPolicyFromContext(ctx)
	cfg.DisabledTools = append([]string(nil), cfg.DisabledTools...)
	if cfg.BlockedActions != nil {
		blocked := make(map[string][]string, len(cfg.BlockedActions))
		for k, v := range cfg.BlockedActions {
			blocked[k] = append([]string(nil), v...)
		}
		cfg.BlockedActions = blocked
	}
	return context.WithValue(ctx, pluginPolicyKey{}, pluginCallPolicy{Config: cfg, Scope: scope, UserRequest: request, Depth: previous.Depth})
}

func pluginPolicyFromContext(ctx context.Context) (pluginCallPolicy, bool) {
	if ctx == nil {
		return pluginCallPolicy{}, false
	}
	policy, ok := ctx.Value(pluginPolicyKey{}).(pluginCallPolicy)
	return policy, ok
}

func pluginChildContext(ctx context.Context) context.Context {
	policy, _ := pluginPolicyFromContext(ctx)
	policy.Depth++
	return context.WithValue(ctx, pluginPolicyKey{}, policy)
}
