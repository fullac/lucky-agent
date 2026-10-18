# LuckyAgent core

The public Go module `github.com/yurika0211/lucky-agent/agent` owns Agent
execution: the loop, context selection and packing, tool/approval policy,
session-aware execution, memory-use rules, convergence and streaming events.
It has its own `go.mod` so the application can depend on it independently of
the repository's executable launcher.

Backends, model protocols, session storage, RAG, configuration and default
resource preparation come from `luckyagent-runtime`. The full product's CLI,
HTTP/channel commands and embedding setup come from `luckyagent-app`. This core
module never imports the application module.

## Explicit injection

```go
resources, err := bootstrap.Open(manager, nil) // runtime/bootstrap
if err != nil {
    return err
}
instance, err := agent.NewWithDependencies(agent.Options{TakeOwnership: true}, resources)
if err != nil {
    _ = resources.Close()
    return err
}
defer instance.Close()
```

`NewWithDependencies` binds already-prepared dependencies. It does not create
backend resources, start MCP/proactive/heartbeat services, or write files.
Obtaining `StartupHooks` also has no activation side effects. Applications may
explicitly call `resources.Start(ctx, instance.StartupHooks())` when they want
the full configured product lifecycle.

Resources remain caller-owned unless `TakeOwnership` is true; ownership transfers
only on successful construction. Closing an owned instance closes the prepared
resource bundle once. Closing a borrowed instance leaves that bundle open.

`New` and `NewWithMemory` remain compatibility convenience constructors for
the default application lifecycle. New embeddings should use explicit
dependencies; full applications can use `luckyagent-app/bootstrap.New`.
The existing shared `config.Manager` and concrete public runtime types remain
compatible inputs; this migration does not invent a second configuration schema.

## Validate independently

```bash
GOWORK=off go test -mod=readonly -race ./...
GOWORK=off go vet ./...
```

The parent directory's `go test ./...` skips this nested module. Root `make test`
and CI run both modules explicitly. The import-boundary test prevents this core
from depending on the application or former host-internal paths.
