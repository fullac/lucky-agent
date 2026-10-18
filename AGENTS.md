# LuckyAgent Agent core and launcher

This repository primarily maintains the public Agent core plus a thin `lh`
launcher. Application commands and runtime backends live in independent modules.

## Modules and ownership

- `agent/` is module `github.com/yurika0211/lucky-agent/agent`: Agent loop,
  context planning/packing, tool and approval policy, memory-use rules,
  session-aware/durable execution, convergence and stream events.
- The root is module `github.com/yurika0211/lucky-agent`: `cmd/la/main.go`,
  build/release entrypoints and product documentation. It imports the external
  CLI and still builds the complete `lh` executable.
- `github.com/yurika0211/luckyagent-app`: CLI/REPL, API and channel product
  handlers, shared application adapters, full embedding SDK and benchmarks.
- `github.com/yurika0211/luckyagent-runtime`: provider/model protocols,
  configuration, sessions, memory backend, RAG/embedding, MCP and support
  components. It does not import the core or application.
- Existing tools/API/gateway modules retain generic execution and wire/platform
  implementation. The Agent leaf module must never import luckyagent-app.

The Agent leaf module and root launcher have separate manifests to avoid a
module dependency cycle: launcher -> app -> Agent -> runtime/tools. Use actual
published commit versions; never commit machine-specific replacements.

## Construction and lifecycle

`agent.NewWithDependencies(Options, *Dependencies)` binds prepared runtime
components without backend construction, network startup or file writes.
`Options.TakeOwnership` transfers cleanup only on successful construction.
`StartupHooks` is inert until an application activates it. Default applications
use `luckyagent-app/bootstrap.New`; legacy core New/NewWithMemory remain
compatibility constructors. Shared config.Manager and public runtime storage
contracts retain their existing schema and semantics.

## Context rules

- Root AGENTS.md/agents.md is project context; runtime manuals also come from
  description/AGENTS.md, description/agents.md and legacy manual paths.
- Preserve the latest user turn and its following messages when fitting history.
- Retrieved memory/RAG are prior evidence, not the current task. Explicit current
  user intent takes precedence. Sessions, durable memory and RAG stay distinct.
- Inspect agent/context_planner.go, system_prompt.go, memory_gate.go and the
  external session/channel adapters when investigating contamination.

## Product compatibility

The runtime home remains ~/.luckyagent. CLI commands, /api/v1 HTTP routes,
SSE/WS/gRPC fields, gateway protocols and session JSON/blob/compact formats are
unchanged by moving source. Check route handlers in luckyagent-app/apiapp and
transport catalogs in luckyagent-api before documenting endpoints.

Configuration schema/defaults live in luckyagent-runtime/config. The example
and GitHub Pages source remain here. Whenever a field or feature changes,
compare config.example.json and update the corresponding product documentation.

## Workflow

1. Inspect git status --short; preserve unrelated modifications.
2. Read the smallest relevant code path and tests. Use rg and apply_patch.
3. Run focused checks, or both module suites for cross-cutting changes:

```bash
(cd agent && GOWORK=off go test -mod=readonly -race ./...)
GOWORK=off go test -mod=readonly ./...
GOWORK=off go build -mod=readonly -o lh ./cmd/la
```

Parent ./... skips the nested Agent module. `make test` and CI explicitly check
both. The committed portable go.work selects the root and ./agent, so normal
builds and Docker use the checked-out Agent source. Optional go.local.work files
may select external app/runtime checkouts with an explicit GOWORK path; keep
machine-specific paths out of go.work. Validate published pins with GOWORK=off.

Report exact verified results and remaining limits. Do not claim tests,
publication, merge or deployment succeeded without direct evidence. Project
findings and decisions are synchronized to the user's Obsidian project vault.
