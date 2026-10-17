# 记忆系统解耦研究：直接接入 Aestus

研究基线：`master` 的 `b2df9cc`，分支 `research/memory-decoupling`，2026-10-17。本文件是研究提案，尚未实施功能重构。MCP PR #292 仍开放，未把该 PR 或其他未合并分支当作主分支现状。

目标已调整为：**LuckyAgent 通过 Go module 直接依赖 Aestus，由薄适配器接入；记忆引擎的演进归 Aestus。** LuckyAgent 只保留记忆使用场景、准入与权限、上下文组装及工具执行。之前“先拆内部仓储/召回引擎”的方向不再作为最终实施方案。

## Aestus 已有能力与 SDK 状态

2026-10-17 核对 [Aestus main `468db80`](https://github.com/yurika0211/aestus/tree/468db804c97fc2f043efe4836a35f0c12c663ba6)：

- [README](https://github.com/yurika0211/aestus/blob/468db804c97fc2f043efe4836a35f0c12c663ba6/README.md) 已公开 `github.com/yurika0211/aestus/memory`，明确允许 LuckyAgent 作为普通 Go module 使用。它已是库式接入入口；“等 SDK”应理解为等接口与版本契约稳定，不能假定将来一定是远程客户端。
- 当前可调用 `NewStore`、`SaveWithOptionsResult`、`SearchWithOptions`、`RouteWithOptions` 和 `Close`；短期缓存、中期摘要、维护、策略、Tidal 已有分包。这些是当前 API，以下 LuckyAgent 端口是提案，两者不可混称。
- GitHub tags 与 issues 查询均为空；本次未找到正式 SDK 发布版本或承诺时间。接入时锁定经验证的 tag；没有 tag 时锁 commit，由 Go 记录 pseudo-version 与校验和，避免追踪浮动 main。
- [Aestus go.mod](https://github.com/yurika0211/aestus/blob/468db804c97fc2f043efe4836a35f0c12c663ba6/go.mod) 声明 Go 1.27.0，LuckyAgent 当前声明 Go 1.25.0；实际接入前必须决定兼容版本或协调工具链升级。适配器不能消除上游编译依赖，Tidal/SQLite 的依赖也需检查实际 import 图。

优先直接引用公开 Go 包。只有 SDK 最终要求独立服务，或出现跨进程共享需求时，才增加 HTTP adapter；目前不为此新建服务或设计未存在的远程 API。

## 先分清四类数据

| 数据 | 当前所有者 | 建议边界 |
| --- | --- | --- |
| 原始会话与恢复状态 | `internal/session` | 保持会话事实源，不自动等同于可复用事实 |
| 短期窗口与压缩摘要 | `memory.SessionShortTermStore` / `ShortTermBuffer` | 会话状态生命周期；消息转换留在宿主适配器 |
| 可检索会话摘要 | `memory.MidTermStore` | 独立摘要读写端口，保留 session 来源及后续可见性约束 |
| 可复用事实、偏好、规则 | `memory.Store` 的 Markdown vault | 通过 Aestus adapter 交给上游记忆引擎 |

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
| `memory/tidal_store.go` | 可选 Tidal SQLite 实现在同一 Go 包中，禁用功能也不会移除包级编译依赖 | 随引擎复用 Aestus 的实现；不在 LuckyAgent 再维护一套 Tidal |

以上是源码事实，不等于已证明存在跨用户数据泄露。当前可见性依赖调用路径；需要统一契约，并以不同身份、相同查询的契约测试验证。

### 写入用例值得先收口

`Store.SaveWithOptionsResult` 会清理生成的截断标记并规范化策略，但工具层的 `rejectDirtyMemoryContent` 还执行另一组内容检查。HTTP、自动捕获与工具的分类默认值也不同。不能把“提取接口”和“改变这些语义”混成一次无说明的结构重构。

`HandleRemember` 的 `upsert_state` 按 `ActiveStateIDs → SaveWithOptionsResult → SupersedeEntries` 编排，三步没有共同事务边界。并发时可能留下多个 active 版本，这是待复现的候选问题，尚未做压力测试；进程内串行化也不等于跨文件、跨进程或崩溃下的原子性。

另一个明确的顺序问题是 `format` 校验位于保存之后：输入格式不合法时，调用可能返回错误，但记忆已保存。呈现参数应在调用写入用例前完成校验，或由 adapter 独立处理。

### 召回不应承担模型执行

现有 `Entry`、`SearchResult`、`ActivationExcerpt`、`RouteAnalysis` 已提供结构化数据，`ActivationReranker` 也已有扩展点，可以复用。记忆层不应返回 `provider.Message` 或执行具体工具。

`route_policies` 的数据格式可以保留。当前策略求值是类型化的数据匹配，没有直接运行工具；真正的 `PolicySafe`、禁用工具核验和 provider.ToolCall 构造位于 `agent/memory_gate.go`。这个执行边界应保留。是否进一步抽出独立策略解释器，取决于第二个消费者是否需要它，不先造通用规则平台。

## 推荐边界：宿主用例 → Aestus adapter → 上游 SDK

```text
Agent / 工具 / HTTP
       │
       ▼
LuckyAgent 记忆用例：捕获意图、写入准入、可信作用域
       │ Reader / Writer；只保留消费者需要的契约
       ▼
internal/memoryaestus：SDK 调用、参数/错误/证据映射
       │ Go module，锁定版本
       ▼
Aestus：存储、去重、图召回、时态解析、维护 → Markdown vault

证据沿调用链返回 → LuckyAgent 上下文预算、provider 消息、工具门禁。
会话历史与 RAG 保持各自事实源；缓存/摘要可复用 Aestus，但生命周期归宿主。
```

建议把原 `internal/memory` 收缩为宿主用例和小契约，新增 `internal/memoryaestus` 作为唯一引入 Aestus 的生产包。装配根创建/关闭 adapter，并把 Reader、Writer 注入消费者；Agent、tool、server、context planner 不直接创建上游 Store，也不穿透 adapter 访问 SDK。

以下是候选端口，不是 Aestus 现有或已承诺的 SDK 方法：

```go
type MemoryReader interface {
    Recall(context.Context, RecallRequest) (RecallResult, error)
}

type MemoryWriter interface {
    Remember(context.Context, RememberRequest) (WriteResult, error)
}
```

按实际消费者定义最小 DTO：RecallRequest 携带查询意图、可信作用域、时间和条数；RecallResult 返回证据正文/摘录、ID、来源路径与 heading/block 定位、必要时态信息及策略信号；RememberRequest 携带已校验的内容、分类、保留策略与来源；WriteResult 返回上游 ID、引用和 created/updated 结果。不要把 Aestus 的所有配置和数据结构翻译成另一套通用 SDK。

小契约可以复用 LuckyAgent 已有的中立值模型；不能用 `type Entry = aestus.Entry` 将上游类型重新导出到整个运行时。完整元数据由 Aestus 持久化，adapter 不写 Markdown、不重算排名、不自行推断时态。诊断/管理视图确需的字段通过专用投影增加，不把每个 SDK 方法镜像成接口。

### 职责如何分配

| LuckyAgent 保留 | Aestus 提供 | 薄适配器负责 |
| --- | --- | --- |
| 何时抽取、用户是否要求记住、内容准入、当前身份与可见范围 | Markdown 读写、格式、去重、索引、图召回/重排、时态解析、维护算法 | 将宿主意图和作用域映射到实际支持的 SDK 调用 |
| 会话恢复/清理、摘要触发与 LLM 调用、RAG 资料检索 | 可复用的短期缓冲、中期摘要存储等库组件 | 按需封装组件；不复制算法到 LuckyAgent |
| token 预算、provider.Message、最新用户指令优先、工具审批与执行 | 结构化证据、定位信息、可支持的策略求值结果 | 返回中立证据与信号，保留来源；不执行模型或工具 |
| 运行时装配、停机、配置、错误呈现、观测 | 实际支持的持久化/维护/关闭能力 | 管理 SDK 生命周期、版本差异和错误映射 |

首次只迁持久事实读写与召回。现有会话缓存可暂留，但随后按需复用 Aestus 分包，避免长期保留同源实现；即使使用库中的缓存，session ID、清理时机和摘要触发仍由 LuckyAgent 决定。

### SDK 接入前必须验证的边界

1. **作用域必须在召回前生效。** `turn.Scope` 由宿主身份映射，不由模型自由填写 namespace。当前 Aestus `RouteOptions` 只有 EntryFilter，`RouteWithOptions` 先 Activate 再过滤，SearchOptions 也没有 namespace。不能用最终结果过滤宣称图传播、top-K 或状态解析已隔离。需要上游提供候选前作用域能力，或由 adapter 选择按可信范围物理隔离的 Store/vault；共享记忆的组合检索与排序需要单独验证，不能把标签当权限。
2. **状态更新不能假称原子。** 当前仍是 ActiveStateIDs / SaveWithOptionsResult / SupersedeEntries 等独立方法。需要的原子 upsert、失败恢复和幂等契约应在 Aestus 提供与测试。LuckyAgent 不新写一套状态引擎，也不把三次调用或进程内 mutex 包装成跨进程/崩溃安全事务。迁移期间保留旧语义必须明确标注；要求原子保证的操作在能力缺失时返回不支持。
3. **取消与错误遵循实际能力。** 当前这些 Store 方法没有 context.Context，有的召回只返回结果不返回 error。宿主端口可以接受 context，但 adapter 只能保证调用前取消检查，不能假装能中断 SDK 内部读写，也不能用超时 goroutine 掩盖已发生的写入。禁止自动重试不确定是否已提交的写入。
4. **只读与反馈要显式。** 当前 Search 可更新访问统计。交互召回、诊断查询和反馈写入需区分，不把一次 GET/检查默认为无副作用。上游缺少的能力按真实需求逐项补齐，不先造能力注册平台。

## 实施顺序与验收

1. **现在：收缩宿主依赖。** 记录工具、HTTP、自动捕获、context、CLI/管理入口的行为，定义消费者端口。必要时用现有 Store 的临时 adapter 验证边界，但不继续拆建内部记忆引擎。本次仅改研究文档，尚未创建这些代码。
2. **SDK 可接入时：只实现 Aestus adapter。** 锁定版本，解决 Go 工具链兼容，在临时 vault 对真实 SDK 跑契约测试；先覆盖写入→召回→证据进入 context 的完整用例，再覆盖 core memory、状态更新、维护及管理路径。功能缺口优先补 Aestus。
3. **整体切换：一条读写链。** 备份实际 vault，在副本验证格式和读写兼容后，保留现有 `${home}/memory` 路径切换所有入口。删除旧 Store/graph/tidal 等重复实现；兼容方法只能转发。不要做生产双写，也不要在 Aestus 故障时静默回退写入旧引擎。
4. **后续升级：适配器吸收变化。** 将来正式 SDK 改 import path 或调用方式时，预期改动集中于 adapter 和装配根，并重跑契约测试。SDK 的实际语义变化仍需审查，不能承诺所有升级对宿主完全无感。

接入可先沿用现有 `memory` 配置，由装配根映射；没有第二个生产 backend 时，不新增 `backend=legacy|aestus` 等长期切换开关。将来确有远程 SDK 再增加真实需要的地址/凭据配置。

关键验收不是行数，而是：

- 不创建 Agent、不配置 LLM，也能对真实 Aestus adapter 的临时 vault 保存、召回、检查支持的状态更新语义并关闭。Fake 仅验证宿主编排，不能代替 SDK 契约测试。
- 同一有效请求经不同入口得到一致领域结果；呈现失败不会造成“错误回执但已写入”的隐式副作用。
- 不同作用域在图传播、top-K 和状态解析时保持一致，不能只在最终列表过滤。
- 保留笔记 ID、相对路径、wikilink、heading/block 定位、历史记录和可重建索引。
- 并发状态更新、写入失败、重试、只读检查及会话清理有明确契约测试。
- SDK 不可用时区分“记忆不可用”和“查无记忆”；显式 remember 不报成功，自动捕获失败可观察。是否继续生成回复属于宿主策略，不触发另一套持久化。
- 生产 Aestus import 只出现在适配包；核心运行时不依赖 SDK DTO，旧引擎没有剩余生产调用路径。升级前后用相同 fixture 检查语义兼容。

## 与相邻工作的衔接

本地已有 `cleanup/delta-memory-state-20261017`（`9260856`）处理暂态记忆更新与旧事实注入，基线比较涉及工具写入和 context planner。本研究未合入或修改该分支。后续实施应先确认它的合并状态，复用其语义和用例，避免再写另一套暂态状态推断。

`docs/memory/README.md` 包含历史整理，部分概述不能代替本次源码核验，例如当前短期缓存已经按 session 管理。

## 外部实践及取舍

LangGraph 将跨会话 Store 与线程级 checkpointer 分开，并用 namespace 寻址记忆；这支持分离会话状态与事实存储的方向。namespace 本身不替代宿主授权。参见 [官方 Store 文档](https://github.com/langchain-ai/docs/blob/main/src/oss/langgraph/stores.mdx)。

Google ADK 将 Session/State 与 MemoryService 区分，并由 Runner 注入服务。可借鉴服务边界；其部分实现保存完整会话的行为，不直接套用到 LuckyAgent 的事实写入准入。参见 [官方 MemoryService 文档](https://github.com/google/adk-docs/blob/main/docs/sessions/memory.md)。

该服务边界用于接入 Aestus，不意味着 LuckyAgent 另建通用记忆平台。SDK 适配包不放在 tools 中：工具只是入口之一，自动召回、HTTP 和维护也必须能使用同一后端。
