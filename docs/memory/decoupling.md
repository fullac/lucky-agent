# 记忆系统解耦研究

研究基线：`master` 的 `b2df9cc`，分支 `research/memory-decoupling`，2026-10-17。本文件是研究提案，尚未实施功能重构。MCP PR #292 仍开放，未把该 PR 或其他未合并分支当作主分支现状。

建议先建立记忆读写的应用边界，再逐步拆分会话状态、仓储和召回实现。现有 `internal/memory` 已有存储、图召回和重排扩展点，单纯再移动目录不能解决上层的规则分散与具体类型依赖。

## 先分清四类数据

| 数据 | 当前所有者 | 建议边界 |
| --- | --- | --- |
| 原始会话与恢复状态 | `internal/session` | 保持会话事实源，不自动等同于可复用事实 |
| 短期窗口与压缩摘要 | `memory.SessionShortTermStore` / `ShortTermBuffer` | 会话状态生命周期；消息转换留在宿主适配器 |
| 可检索会话摘要 | `memory.MidTermStore` | 独立摘要读写端口，保留 session 来源及后续可见性约束 |
| 可复用事实、偏好、规则 | `memory.Store` 的 Markdown vault | 记忆应用服务与可替换仓储 |

RAG 继续负责索引资料检索。`TierShort` 是已落盘记忆条目的层级/衰减标签，不等于易失的短期会话窗口。`Store` 与 `MidTermStore` 目前都使用 vault 下的 `30_Sessions`，但使用不同记录类型；逻辑分离不要求立刻搬迁这些文件。

## 已核验的耦合

| 位置 | 当前事实 | 解耦方向 |
| --- | --- | --- |
| `memory/short_term.go`（GetContext） | 缓存内部已有中立 `ConversationTurn`，但 `GetContext` 返回 `provider.Message`，把整个 memory 包连到 provider 包 | 返回中立快照，将 provider 消息构造移到宿主 |
| `agent/agent.go`（initMemoryRuntime） | 同时装配 durable store、Tidal、短期缓存、中期摘要及 sessions | 装配根管理生命周期，各消费者只拿所需端口 |
| `tool/memory_service.go`（HandleRemember） | 写入准入、分类推断、状态更新编排与文本/JSON 呈现混在工具服务 | 参数解析与呈现留在 adapter，业务写入交给 Writer |
| `agent/agent.go`（saveConversationMemoryFromSession / Remember） | 自动事实抽取、显式 Remember、维护动作仍由 Agent 发起；部分路径忽略保存错误 | 独立捕获/写入用例和可观察的结果，Agent 负责触发 |
| `server/server.go`（handleMemory） | HTTP POST 通过 Agent 的 Remember 方法直达 Store，没有复用工具层的完整准入检查 | 让各入口调用同一写入用例；统一规则属于单独的行为变更 |
| `agent/context_planner.go`（contextRuntime / buildRelevantMemoryMessage） | planner 虽已有 contextRuntime 接口，仍要求具体 `*memory.Store` / `*memory.MidTermStore`，并自己组装证据和策略提示 | Reader 返回类型化证据，planner 保留预算与 provider 呈现 |
| `memory/memory.go`（RouteWithOptions） | 在激活/候选截断后才执行调用方的 `EntryFilter`；工具 SearchOptions 没有同样的作用域端口 | 在候选、图传播、排序与状态解析前建立统一可见集合 |
| `memory/tidal_store.go` | 可选 Tidal SQLite 实现在同一 Go 包中，禁用功能也不会移除包级编译依赖 | 延用已有 ActivationReranker 端口，后续独立 Tidal adapter |

以上是源码事实，不等于已证明存在跨用户数据泄露。当前可见性依赖调用路径；需要统一契约，并以不同身份、相同查询的契约测试验证。

### 写入用例值得先收口

`Store.SaveWithOptionsResult` 会清理生成的截断标记并规范化策略，但工具层的 `rejectDirtyMemoryContent` 还执行另一组内容检查。HTTP、自动捕获与工具的分类默认值也不同。不能把“提取接口”和“改变这些语义”混成一次无说明的结构重构。

`HandleRemember` 的 `upsert_state` 按 `ActiveStateIDs → SaveWithOptionsResult → SupersedeEntries` 编排，三步没有共同事务边界。并发时可能留下多个 active 版本，这是待复现的候选问题，尚未做压力测试；进程内串行化也不等于跨文件、跨进程或崩溃下的原子性。

另一个明确的顺序问题是 `format` 校验位于保存之后：输入格式不合法时，调用可能返回错误，但记忆已保存。呈现参数应在调用写入用例前完成校验，或由 adapter 独立处理。

### 召回不应承担模型执行

现有 `Entry`、`SearchResult`、`ActivationExcerpt`、`RouteAnalysis` 已提供结构化数据，`ActivationReranker` 也已有扩展点，可以复用。记忆层不应返回 `provider.Message` 或执行具体工具。

`route_policies` 的数据格式可以保留。当前策略求值是类型化的数据匹配，没有直接运行工具；真正的 `PolicySafe`、禁用工具核验和 provider.ToolCall 构造位于 `agent/memory_gate.go`。这个执行边界应保留。是否进一步抽出独立策略解释器，取决于第二个消费者是否需要它，不先造通用规则平台。

## 推荐的应用边界

```text
Agent / 工具 / HTTP
       │ 只依赖所需读写端口
       ▼
记忆应用服务：准入、可信作用域、召回、状态更新
       │                     └─ 类型化证据 / 策略信号
       ▼                                     ▼
Markdown 仓储                         上下文呈现 / 工具门禁

会话状态与摘要：独立生命周期，由宿主关联，不并入事实写入接口。
```

以下是候选端口，不是已存在的 API；接口可放在消费方或小契约包，避免新建一个包含所有管理功能的大接口：

```go
type MemoryReader interface {
    Recall(context.Context, RecallRequest) (RecallResult, error)
}

type MemoryWriter interface {
    Remember(context.Context, RememberRequest) (WriteResult, error)
}
```

请求需要明确的查询/写入意图、由宿主绑定的记忆命名空间、时间与限额。聊天 `turn.Scope` 可以由 adapter 映射，但不应直接充当完整授权；模型参数不能自行选择另一个用户的命名空间。管理员查看非 active 记录应是显式管理能力。

结果尽量复用现有中立 Entry、摘录、来源引用、时态状态与保存结果，不另建一套同义数据模型。上下文 token 预算、消息角色、工具审批和执行继续归宿主。召回是否更新访问计数、是否写反馈要显式，便于检查接口保持只读。

Writer 内部协调原子状态更新；仓储决定如何实现版本检查、提交失败恢复与持久化。Markdown 继续是事实源，图索引与 Tidal 数据保持派生状态，不能引入第二个不一致的权威副本。

第一阶段可以让现有 Store 支撑这些端口。随后再按需要分离 RecallEngine、VaultRepository 和 Tidal adapter。单独包/模块发布需要检查整个 Go import 图：只把消费者字段改成接口，还不能移除同包的 provider、SQLite 依赖。

## 实施顺序与验收

1. **记录兼容契约并建立端口**：覆盖 HTTP、工具、自动捕获的现有差异；去掉短期缓存的 provider 消息转换依赖；先迁移一个实际消费者验证边界。
2. **统一写入与查询语义**：收拢准入、分类、状态更新、作用域和错误回传。规则变化单独列出，包括 HTTP/自动捕获之前能保存而之后会拒绝的内容，避免假称无行为变化。
3. **分离生命周期与实现**：会话摘要/维护通过中立事件或明确方法触发；按需拆出 Tidal 与仓储适配器。只保留一条生产读写链，兼容方法只转发。

关键验收不是行数，而是：

- 不创建 Agent、不配置 LLM，也能对临时 vault 保存、召回、状态替换和关闭。
- 同一有效请求经不同入口得到一致领域结果；呈现失败不会造成“错误回执但已写入”的隐式副作用。
- 不同作用域在图传播、top-K 和状态解析时保持一致，不能只在最终列表过滤。
- 保留笔记 ID、相对路径、wikilink、heading/block 定位、历史记录和可重建索引。
- 并发状态更新、写入失败、重试、只读检查及会话清理有明确契约测试。

## 与相邻工作的衔接

本地已有 `cleanup/delta-memory-state-20261017`（`9260856`）处理暂态记忆更新与旧事实注入，基线比较涉及工具写入和 context planner。本研究未合入或修改该分支。后续实施应先确认它的合并状态，复用其语义和用例，避免再写另一套暂态状态推断。

`docs/memory/README.md` 包含历史整理，部分概述不能代替本次源码核验，例如当前短期缓存已经按 session 管理。

## 外部实践及取舍

LangGraph 将跨会话 Store 与线程级 checkpointer 分开，并用 namespace 寻址记忆；这支持分离会话状态与事实存储的方向。namespace 本身不替代宿主授权。参见 [官方 Store 文档](https://github.com/langchain-ai/docs/blob/main/src/oss/langgraph/stores.mdx)。

Google ADK 将 Session/State 与 MemoryService 区分，并由 Runner 注入服务。可借鉴服务边界；其部分实现保存完整会话的行为，不直接套用到 LuckyAgent 的事实写入准入。参见 [官方 MemoryService 文档](https://github.com/google/adk-docs/blob/main/docs/sessions/memory.md)。

独立 HTTP 服务适合出现跨进程共享与运维需求后再评估；目前优先做进程内边界，保留未来服务 adapter 的出口。暂不替换数据库、重写评分算法或迁移用户 vault。
