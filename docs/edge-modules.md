# Messaging and API module boundaries

The messaging and API transport now have independent Go module sources:

- `github.com/yurika0211/luckyagent-gateway`: platform connections, message and
  media presentation, collector, and neutral `turn` contracts.
- `github.com/yurika0211/luckyagent-api`: HTTP policies and routes, SSE framing,
  WebSocket connections, run storage/replay, resource controllers, health, and
  generated gRPC protocol.

LuckyAgent owns application behavior and composes both libraries in process.
The new modules do not import LuckyAgent or instantiate an Agent.

| Host package | Ownership |
| --- | --- |
| `internal/gateway/*` | Compatibility aliases for existing imports |
| `internal/server` | Compatibility aliases for API application construction |
| `internal/websocket` | Compatibility aliases for wire types and the host handler |
| `internal/turn` | Neutral contract aliases; durable execution stays application-owned |
| `internal/channelapp/telegram`, `qqofficial` | LuckyAgent commands, model/session/config operations and chat presentation policy |
| `internal/channelapp/websocket` | Agent run execution, cancellation and approval integration; run records/replay come from the API module |
| `internal/apiapp` | Agent/domain operations, storage/configuration policy and resource lifecycle assembly; shared resource parsing/rendering lives in `luckyagent-api/resources` |
| `api/grpc` | Application gRPC implementation and protocol aliases |

This extraction moves reusable transport implementations. It does not describe
all code formerly under `gateway` and `server` as transport: domain operations
continue to live in the application packages above. There is no duplicated
platform implementation or protobuf schema in LuckyAgent.

The v0.2.1 cleanup also moves Telegram command/tool/Skill presentation and
reasoning/progress cards to `luckyagent-gateway/telegram/presentation`. Gateway
session administration and model/config changes remain host policy. Shared
NapCat group-history values live in `luckyagent-tools/chat` and are aliased by
the platform adapter.

`luckyagent-api/resources` owns task/tree/feedback, approvals, cron, autonomy,
Skill lifecycle/installation and collaboration request controllers. It consumes
public tool-runtime ports, with callbacks for host-owned installation policy
and orchestrator session creation. `internal/apiapp/resource_controller.go` is
the assembly point; it normalizes an absent Agent before interface injection.

The local, ignored `go.work` expects the two module checkouts next to LuckyAgent:

```bash
go work init . ../luckyagent-gateway ../luckyagent-api ../luckyagent-tools
go work edit -replace=github.com/yurika0211/luckyagent-gateway@v0.2.1=../luckyagent-gateway
go work edit -replace=github.com/yurika0211/luckyagent-api@v0.2.1=../luckyagent-api
go work edit -replace=github.com/yurika0211/luckyagent-tools@v0.2.1=../luckyagent-tools
```

These workspace replacements keep unpublished versions local. The committed
`go.mod` contains no local replacement. After both repositories and release tags
exist, verify the released modules with `GOWORK=off` from a clean checkout before
treating publication as done.

Validation:

```bash
# In each independent module
go test ./...

# In LuckyAgent
go test ./internal/apiapp ./internal/channelapp/... ./api/grpc ./internal/cli/lhcmd ./internal/agent
go test ./...
```

No new configuration key is introduced. Existing server, gateway and attachment
configuration remains in `config.example.json`. HTTP/WebSocket/gRPC protocol
names and platform permission checks retain their previous behavior.
