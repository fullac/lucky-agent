# 独立插件模块

仓库：<https://github.com/yurika0211/luckyagent-plugins>。
Go 模块 `github.com/yurika0211/luckyagent-plugins` 只使用标准库，负责 manifest、
marketplace、安装快照、版本与信任。公开 Agent 的 `agent/plugin_runtime.go` 将插件
接入 Skills、命令、Agent 子会话、MCP 和工具 Hooks；模型和权限仍由宿主管理。

```bash
lh plugin validate /path/to/luckyagent-plugins/examples/starter
lh plugin install /path/to/luckyagent-plugins/examples/starter
lh plugin list
lh plugin show starter --json
lh plugin search review --marketplace /path/to/luckyagent-plugins/marketplace.json
lh plugin install https://github.com/team/plugins.git --ref v1.0.0 --subdir plugins/reviewer --trust
lh plugin disable starter
lh plugin enable starter
lh plugin update starter
lh plugin remove starter
```

管理命令不构造 Agent、不运行脚本、不连接 MCP。已运行 API / 网关需重启才使用新
快照。存储为 `${runtimeHome}/plugins/index.json` 和 `bundles/<name>/<sha256>`；
不修改原有 skills 目录。本次没有新增 config.json 字段，config.example.json 无需新增键。
卸载移除登记但保留缓存，避免运行中的进程丢失文件；首版不自动 GC。

只有明确 `--trust` 的快照能启动 MCP/Hooks 或注册技能脚本。更新不继承旧内容信任，
保留此前手动禁用状态。未授信的纯指令插件可以使用，技能脚本不会注册为执行工具。

插件技能以 `plugin:skill` 合入 `skill_read` 与路由；发现过程不执行 --help。
`/starter:review <arguments>` 在 CLI、HTTP 聊天和网关走正常 Agent Loop，参数按字面
展开 `$ARGUMENTS`，保留参数大小写、连字符、网关回复上下文与投递要求，以及聊天输入的会话和附件。`plugin_info`
列出当前插件、命令和 Agent 预设。
`plugin_agent(name="starter:reviewer", task="...")` 在独立子会话运行，继承父来源、
记忆范围、模型、工具/动作限制和取消上下文；不复制父历史，不递归委派，批准启动不等于
自动批准子会话之后的每个工具调用。

MCP 服务以 `plugin.server` 命名，复用现有连接生命周期，不覆盖同名主机配置。
插件脚本/MCP 遵循宿主审批，在 iso 模式禁用。脚本采用可取消子进程，最长60秒或更短
父限制、最多64KiB输出，参数通过 SKILL_ARG_* 环境变量传递。支持的 Unix 平台使用独立
进程组，在取消和调用完成后清理后台子进程。Windows 尚未实现进程树清理，首版拒绝
插件脚本执行；纯指令插件和插件管理不受此限制。

原生工具 Hooks 使用 Lucky Payload/Decision 协议，支持 PreToolUse/PostToolUse。
插件 Hooks 与用户 Hooks 合并并采用 fail-closed；禁用的用户 Hooks 不会被顺带启用。
插件 Hooks 不受独立的用户 hooks.enabled 开关控制。
Hooks 沿用宿主独立超时，不随聊天取消即时中断；Hook 改写工具参数后仍会重新检查当前
工具与动作限制。插件执行信任不提供操作系统沙箱。

兼容入口为 .lucky-plugin/plugin.json、根 plugin.json、.codex-plugin/plugin.json
和 .claude-plugin/plugin.json 的可转换子集。外部 Hook 协议、LSP、App OAuth、Agent
模型/权限覆盖、依赖解析、签名、项目安装范围、热卸载和管理UI尚未实现；未实现执行
元数据会明确拒绝。这是可执行 v0.1 基础，不代表完整 Codex/Claude Code 兼容。

## HTTP 命令入口

接口遵循现有 API 鉴权。`GET /api/v1/commands` 返回内置命令和运行时启动时加载的
插件命令；插件条目使用 `group: "plugins"`，`name` 是完整的 `starter:review`
这类命名空间标识。客户端应原样传回名称，插件命令不会经过内置命令的小写化或
连字符替换。

`POST /api/v1/commands` 可以提交：

```json
{"command":"starter:review","args":"UI/HTTP-API.go --Keep-Case","session_id":"existing-session-id"}
```

`command` 可带一个前导 `/`。接口展开提示词后调用真实 Agent Loop，采用宿主配置的
Agent 限制和审批策略；省略 `session_id` 时创建会话，指定不存在的会话返回 `404`。
执行结果为 `{"command":"starter:review","ok":true,"output":"...","session_id":"..."}`；
Loop 失败仍返回 HTTP `200`，但 `ok` 为 `false`，`output` 是错误说明。

`POST /api/v1/chat/sync` 可把完整命令放进 `message`，例如
`{"message":"/starter:review UI/HTTP-API.go --Keep-Case"}`。已加载的插件命令会先于
内置 slash 路由识别，继续正常聊天 Loop；此入口支持会话、附件和现有聊天请求选项，
返回标准 `ChatResponse`。安装和启用状态改变后仍需重启 API 实例。
完整字段和响应见 [API 文档](API.md#commands)。

独立仓库 docs/format.md、docs/install.md、docs/cli.md 与 docs/architecture.md
包含字段和边界。联合开发用忽略的 go.local.work 选择 app/runtime；主仓自带 go.work 只选择根模块和 ./agent。正式 go.mod 固定可下载版本。

```bash
# 独立插件模块
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
# 主仓 Agent 子模块
cd /path/to/lucky-agent/agent
GOWORK=off go test -race ./... -run 'TestPlugin|TestInjected'
# 应用模块
cd /path/to/luckyagent-app
GOWORK=off go test -race ./apiapp ./cli/lhcmd -run TestPlugin
```
