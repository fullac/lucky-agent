package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/yurika0211/luckyagent-runtime/bootstrap"
	"github.com/yurika0211/luckyagent-runtime/hook"
	"github.com/yurika0211/luckyagent-runtime/session"
	"github.com/yurika0211/luckyagent-runtime/tool"
)

// Options controls core execution resources. Configuration schema and backend
// defaults remain in the runtime module; callers supply prepared components.
type Options struct {
	ContextCacheSize int
	TakeOwnership    bool
}

// Dependencies is the prepared component contract shared with application
// builders. The runtime module owns concrete implementations and defaults.
type Dependencies = bootstrap.Components

func validateDependencies(deps *bootstrap.Components) error {
	if deps == nil {
		return fmt.Errorf("agent components are required")
	}
	checks := []struct {
		name    string
		missing bool
	}{
		{"configuration", deps.Config == nil},
		{"provider", deps.Provider.Provider == nil},
		{"sessions", deps.Sessions == nil},
		{"memory store", deps.Memory.Store == nil},
		{"session memory", deps.Memory.ShortTerm == nil},
		{"memory summaries", deps.Memory.Summaries == nil},
		{"tools", deps.Support.Tools == nil},
		{"tool services", deps.Support.ToolServices == nil},
		{"delegate runtime", deps.Support.DelegateMgr == nil},
		{"autonomy runtime", deps.Support.AutonomyKit == nil},
	}
	for _, check := range checks {
		if check.missing {
			return fmt.Errorf("agent dependency %s is required", check.name)
		}
	}
	return nil
}

// StartupHooks supplies callbacks to the application's explicit lifecycle.
// Obtaining hooks does not activate any service.
func (a *Agent) StartupHooks() bootstrap.Hooks {
	return bootstrap.Hooks{
		LoadSkills:        a.LoadSkills,
		InstallCronEvents: a.installCronEventHandler,
		RestoreCron:       a.restoreCronJobs,
		SaveCron:          a.saveCronJobs,
		StartHeartbeat:    a.initHeartbeatService,
		ResumeAutonomy:    a.StartAutonomyNow,
	}
}

// NewWithDependencies binds prepared resources to an execution coordinator.
// It performs no backend construction and starts no background services.
// Ownership transfers only on success when TakeOwnership is set.
func NewWithDependencies(opts Options, deps *Dependencies) (*Agent, error) {
	if err := validateDependencies(deps); err != nil {
		return nil, err
	}
	cfg := deps.Config
	c := cfg.Get()
	soulRT, providerRT, memoryRT := deps.Soul, deps.Provider, deps.Memory
	ragRT, supportRT := deps.RAG, deps.Support
	sessionManager := deps.Sessions
	proactiveStore, proactiveRuntime := deps.ProactiveStore, deps.ProactiveRuntime
	taskStore, sandboxMgr := deps.Tasks, deps.Sandbox
	cacheSize := opts.ContextCacheSize
	if cacheSize <= 0 {
		cacheSize = 64
	}
	a := &Agent{
		cfg:                 cfg,
		components:          deps,
		ownComponents:       opts.TakeOwnership,
		mcpClient:           deps.MCP,
		soul:                soulRT.Soul,
		tmplMgr:             soulRT.TmplMgr,
		provider:            providerRT.Provider,
		registry:            providerRT.Registry,
		catalog:             providerRT.Catalog,
		tokenStore:          providerRT.TokenStore,
		memory:              memoryRT.Store,
		shortTerms:          memoryRT.ShortTerm,
		midTerm:             memoryRT.Summaries,
		sessions:            sessionManager,
		tools:               supportRT.Tools,
		toolServices:        supportRT.ToolServices,
		gateway:             supportRT.ToolGateway,
		hitl:                newHITLGate(),
		hooks:               hook.NewRunner(buildHookRuntimeConfig(c)),
		msgGateway:          deps.Gateway,
		delegate:            supportRT.DelegateMgr,
		contextWin:          supportRT.ContextWin,
		contextEst:          supportRT.ContextEst,
		ragManager:          ragRT.Manager,
		ragPersist:          ragRT.Persist,
		streamIndexer:       ragRT.StreamIndexer,
		embedderReg:         ragRT.EmbedderReg,
		collabReg:           deps.CollabRegistry,
		collabMgr:           deps.CollabManager,
		metrics:             supportRT.Metrics,
		proactiveStore:      proactiveStore,
		proactiveRuntime:    proactiveRuntime,
		cronEngine:          supportRT.CronEngine,
		cronStore:           deps.CronStore,
		autonomy:            supportRT.AutonomyKit,
		computerMgr:         supportRT.ComputerMgr,
		sandbox:             sandboxMgr,
		contextCache:        newContextMessageCache(cacheSize),
		mediaProcessor:      supportRT.MediaProcessor,
		taskStore:           taskStore,
		autoCompactFailures: make(map[string]int),
		activeModel:         c.Model,
		activeAPIBase:       c.APIBase,
	}

	if err := a.preparePlugins(); err != nil {
		return nil, fmt.Errorf("prepare plugins: %w", err)
	}
	a.hooks.Reload(a.pluginHookConfig(c))
	if a.mcpClient != nil {
		if err := a.mcpClient.Configure(a.pluginMCPConfig(c.MCP)); err != nil {
			return nil, fmt.Errorf("configure plugin MCP: %w", err)
		}
	}

	supportRT.ToolServices.Cron = tool.NewCronToolService(
		supportRT.CronEngine,
		a.saveCronJobs,
		func(id, mode, command string, metadata map[string]string) func() error {
			return a.buildCronTask(id, cronTaskMode(mode), command, metadata)
		},
	)
	supportRT.ToolServices.Autonomy = tool.NewAutonomyToolService(supportRT.AutonomyKit, func() error {
		return a.StartAutonomyNow(context.Background())
	})
	supportRT.ToolServices.Heartbeat = tool.NewHeartbeatToolService(a.handleHeartbeatTrigger, a.handleHeartbeatStatus)
	supportRT.ToolServices.RegisterCoreTools(supportRT.Tools)
	if err := a.registerPluginTools(); err != nil {
		return nil, fmt.Errorf("register plugin tools: %w", err)
	}

	// v0.38.0: 设置 delegate 的 Agent 执行器，让 delegate_task 真正走 Agent Loop
	delegateCfg := supportRT.DelegateMgr.Config()
	supportRT.DelegateMgr.SetSessionInitializer(func(task *tool.DelegateTask) error {
		sess := sessionManager.Ensure(task.ExecutionSessionID)
		sess.SetTitle("delegate-task")
		sess.SetAgentProfile("subagent", delegatedSubagentSystemPrompt())
		if task.Workspace != "" {
			sess.SetCwd(task.Workspace)
		}
		return sess.Save()
	})
	loop := a.loopRunner()
	sessions := sessionManager
	delegateMgr := supportRT.DelegateMgr
	supportRT.DelegateMgr.SetAgentExecutorWithOptions(func(ctx context.Context, description, contextStr string, options tool.AgentExecutorOptions) (string, error) {
		sessionID := tool.DelegateExecutionSessionID(ctx)
		var sess *session.Session
		if sessionID != "" {
			sess = sessions.Ensure(sessionID)
			sess.SetTitle("delegate-task")
		} else {
			sess = sessions.NewWithTitle("delegate-task")
		}
		sess.SetAgentProfile("subagent", delegatedSubagentSystemPrompt())
		if workspace := tool.ExtractDelegateWorkspace(contextStr); workspace != "" {
			if err := os.MkdirAll(workspace, 0o755); err == nil {
				sess.SetCwd(workspace)
			}
		}
		prompt := description
		if contextStr != "" {
			prompt = fmt.Sprintf("%s\n\nContext: %s", description, contextStr)
		}
		loopCfg := buildDelegateChildLoopConfig(delegateCfg, options.AllowRecursiveDelegate)
		result, err := loop.Run(ctx, sess, TextUserTurnInput(prompt), loopCfg)
		if result != nil {
			lastTool := ""
			if n := len(result.ToolCalls); n > 0 {
				lastTool = result.ToolCalls[n-1].Name
			}
			delegateMgr.RecordExecutionMetrics(tool.DelegateTaskID(ctx), len(result.ToolCalls), lastTool)
		}
		if err != nil {
			return "", err
		}
		return result.Response, nil
	})

	// v0.38.0: 将 executor 注入到已注册工具所绑定的 autonomy 实例，避免启动时替换实例。
	a.autonomy.SetExecutor(&agentExecutorAdapter{loop: loop, sessions: sessions})

	return a, nil
}
