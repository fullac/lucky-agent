// Package tool preserves LuckyAgent imports while implementations live in
// luckyagent-tools and internal/toolapp.
package tool

import (
	toolkit "github.com/yurika0211/luckyagent-tools"
	application "github.com/yurika0211/luckyagent/internal/toolapp"
)

var AnnotateToolCall = toolkit.AnnotateToolCall
var AnnotateToolCallWithTemplates = toolkit.AnnotateToolCallWithTemplates

type ApprovalOption = toolkit.ApprovalOption

var AskUserTool = toolkit.AskUserTool

type AuditEntry = toolkit.AuditEntry
type AuditStats = toolkit.AuditStats

var CSVQueryTool = toolkit.CSVQueryTool
var CalculateTool = toolkit.CalculateTool

type CapabilitySummary = toolkit.CapabilitySummary
type CapabilityTool = toolkit.CapabilityTool

const CatBuiltin = toolkit.CatBuiltin
const CatDelegate = toolkit.CatDelegate
const CatMCP = toolkit.CatMCP
const CatSkill = toolkit.CatSkill

type Category = toolkit.Category

var CheckCollisions = toolkit.CheckCollisions
var CompressOutput = toolkit.CompressOutput
var CopyLocalDir = toolkit.CopyLocalDir
var CurrentTimeTool = toolkit.CurrentTimeTool
var DBSchemaTool = toolkit.DBSchemaTool
var DataQueryTool = toolkit.DataQueryTool
var DataTool = toolkit.DataTool
var DedupOutputs = toolkit.DedupOutputs
var DefaultFilesystemPolicy = toolkit.DefaultFilesystemPolicy
var DefaultInstallLimits = toolkit.DefaultInstallLimits
var DefaultOutputCompressConfig = toolkit.DefaultOutputCompressConfig
var DefaultSandboxConfig = toolkit.DefaultSandboxConfig
var DefaultTierConfigs = toolkit.DefaultTierConfigs
var DocumentReadTool = toolkit.DocumentReadTool

type ErrQuotaExceeded = toolkit.ErrQuotaExceeded
type ErrToolDenied = toolkit.ErrToolDenied
type ErrToolDisabled = toolkit.ErrToolDisabled
type ErrToolNotFound = toolkit.ErrToolNotFound
type ExecutionContext = toolkit.ExecutionContext

var ExtractArchive = toolkit.ExtractArchive
var ExtractDocumentText = toolkit.ExtractDocumentText
var FileDeleteTool = toolkit.FileDeleteTool
var FileListTool = toolkit.FileListTool
var FileMkdirTool = toolkit.FileMkdirTool
var FileMoveTool = toolkit.FileMoveTool
var FilePatchTool = toolkit.FilePatchTool
var FileReadTool = toolkit.FileReadTool
var FileWriteTool = toolkit.FileWriteTool

type FilesystemPolicy = toolkit.FilesystemPolicy

var FormatAuditEntry = toolkit.FormatAuditEntry

type Gateway = toolkit.Gateway
type GatewayResult = toolkit.GatewayResult

var HTTPRequestTool = toolkit.HTTPRequestTool

const InstallAborted = toolkit.InstallAborted
const InstallConfirmed = toolkit.InstallConfirmed
const InstallFailed = toolkit.InstallFailed

type InstallHistory = toolkit.InstallHistory
type InstallLimits = toolkit.InstallLimits
type InstallManager = toolkit.InstallManager
type InstallManagerConfig = toolkit.InstallManagerConfig

const InstallProbing = toolkit.InstallProbing
const InstallReady = toolkit.InstallReady

type InstallRecord = toolkit.InstallRecord

const InstallRejected = toolkit.InstallRejected
const InstallScanning = toolkit.InstallScanning

type InstallSource = toolkit.InstallSource

const InstallStaging = toolkit.InstallStaging

type InstallStore = toolkit.InstallStore
type Ledger = toolkit.Ledger

var LogTool = toolkit.LogTool

type MCPClient = toolkit.MCPClient
type MCPError = toolkit.MCPError
type MCPRequest = toolkit.MCPRequest
type MCPResponse = toolkit.MCPResponse
type MCPServerConfig = toolkit.MCPServerConfig
type MCPToolInfo = toolkit.MCPToolInfo

var MemoryHygieneTool = toolkit.MemoryHygieneTool
var MemoryTool = toolkit.MemoryTool
var MoveDir = toolkit.MoveDir
var NewGateway = toolkit.NewGateway
var NewInstallManager = toolkit.NewInstallManager
var NewInstallStore = toolkit.NewInstallStore
var NewMCPClient = toolkit.NewMCPClient
var NewRegistry = toolkit.NewRegistry
var NewSkillExecutor = toolkit.NewSkillExecutor
var NewSkillLoader = toolkit.NewSkillLoader
var NewSkillRegistry = toolkit.NewSkillRegistry
var NewSkillSandbox = toolkit.NewSkillSandbox
var NewSkillToolService = toolkit.NewSkillToolService
var NewSubscriptionManager = toolkit.NewSubscriptionManager
var NewToolRouter = toolkit.NewToolRouter
var NewTraceRecord = toolkit.NewTraceRecord
var NewTraceRecordWithTemplates = toolkit.NewTraceRecordWithTemplates
var NewUsageTracker = toolkit.NewUsageTracker
var NormalizeRoot = toolkit.NormalizeRoot

type Observation = toolkit.Observation
type OpenCLIConfig = toolkit.OpenCLIConfig

var OpenCLITool = toolkit.OpenCLITool

type OutputCompressConfig = toolkit.OutputCompressConfig

var ParallelCompressOutputs = toolkit.ParallelCompressOutputs

type Param = toolkit.Param

var ParseSubTier = toolkit.ParseSubTier

type PendingApproval = toolkit.PendingApproval

const PermApprove = toolkit.PermApprove
const PermAuto = toolkit.PermAuto
const PermDeny = toolkit.PermDeny

type PermissionLevel = toolkit.PermissionLevel
type Quota = toolkit.Quota

var RAGIndexTool = toolkit.RAGIndexTool
var RAGSearchTool = toolkit.RAGSearchTool
var RecallTool = toolkit.RecallTool

type Rect = toolkit.Rect

var RegisterMCPTools = toolkit.RegisterMCPTools
var RegisterSkillTools = toolkit.RegisterSkillTools

type Registry = toolkit.Registry

var RememberTool = toolkit.RememberTool
var RequestCredentialTool = toolkit.RequestCredentialTool

type RouteRule = toolkit.RouteRule

var SQLQueryTool = toolkit.SQLQueryTool

type SandboxConfig = toolkit.SandboxConfig
type SandboxError = toolkit.SandboxError
type ScanFinding = toolkit.ScanFinding
type ScanReport = toolkit.ScanReport

var ScanSkillDir = toolkit.ScanSkillDir

const SeverityBlock = toolkit.SeverityBlock
const SeverityInfo = toolkit.SeverityInfo
const SeverityWarn = toolkit.SeverityWarn

type ShellContext = toolkit.ShellContext
type SkillCandidate = toolkit.SkillCandidate

const SkillDisabled = toolkit.SkillDisabled
const SkillDiscovered = toolkit.SkillDiscovered

type SkillDocSection = toolkit.SkillDocSection

const SkillEnabled = toolkit.SkillEnabled
const SkillError = toolkit.SkillError

type SkillExecutor = toolkit.SkillExecutor
type SkillInfo = toolkit.SkillInfo

const SkillLoaded = toolkit.SkillLoaded

type SkillLoader = toolkit.SkillLoader
type SkillMetadata = toolkit.SkillMetadata
type SkillReadContent = toolkit.SkillReadContent

const SkillRegistered = toolkit.SkillRegistered

type SkillRegistry = toolkit.SkillRegistry
type SkillSandbox = toolkit.SkillSandbox
type SkillState = toolkit.SkillState
type SkillToolDef = toolkit.SkillToolDef
type SkillToolService = toolkit.SkillToolService

const SkillUnloaded = toolkit.SkillUnloaded
const SkillVersion = toolkit.SkillVersion
const SourceArchive = toolkit.SourceArchive
const SourcePath = toolkit.SourcePath
const SubBasic = toolkit.SubBasic
const SubEnterprise = toolkit.SubEnterprise
const SubFree = toolkit.SubFree
const SubPro = toolkit.SubPro

type SubTier = toolkit.SubTier
type SubscriptionManager = toolkit.SubscriptionManager

var TerminalTool = toolkit.TerminalTool

type TierConfig = toolkit.TierConfig

var ToOpenAIName = toolkit.ToOpenAIName

type Tool = toolkit.Tool
type ToolCallResult = toolkit.ToolCallResult
type ToolRouter = toolkit.ToolRouter
type TraceRecord = toolkit.TraceRecord

var TrialLoad = toolkit.TrialLoad

type TrialOptions = toolkit.TrialOptions
type TrialResult = toolkit.TrialResult

var TruncateOutput = toolkit.TruncateOutput
var TruncateOutputs = toolkit.TruncateOutputs

type UsageRecord = toolkit.UsageRecord
type UsageStats = toolkit.UsageStats
type UsageTracker = toolkit.UsageTracker
type UserSubscription = toolkit.UserSubscription

var ValidSkillName = toolkit.ValidSkillName
var WebFetchTool = toolkit.WebFetchTool

type WebSearchConfig = toolkit.WebSearchConfig

var WebSearchTool = toolkit.WebSearchTool

type AgentExecutorFunc = application.AgentExecutorFunc
type AgentExecutorOptions = application.AgentExecutorOptions
type AgentExecutorWithOptionsFunc = application.AgentExecutorWithOptionsFunc
type ApprovalRequiredError = application.ApprovalRequiredError
type AutonomyStartFunc = application.AutonomyStartFunc
type AutonomyToolService = application.AutonomyToolService
type BuiltinToolService = application.BuiltinToolService
type CodexToolService = application.CodexToolService
type ComputerManager = application.ComputerManager
type ComputerUseConfig = application.ComputerUseConfig
type ComputerUseToolService = application.ComputerUseToolService
type CronSaveFunc = application.CronSaveFunc
type CronTaskFactory = application.CronTaskFactory
type CronToolService = application.CronToolService

var DefaultDelegateConfig = application.DefaultDelegateConfig
var DelegateCancelTool = application.DelegateCancelTool

type DelegateConfig = application.DelegateConfig

var DelegateExecutionSessionID = application.DelegateExecutionSessionID

type DelegateManager = application.DelegateManager
type DelegateTarget = application.DelegateTarget
type DelegateTask = application.DelegateTask

var DelegateTaskID = application.DelegateTaskID
var DelegateTaskTool = application.DelegateTaskTool
var DelegateToMCPTool = application.DelegateToMCPTool
var DelegateToSkillTool = application.DelegateToSkillTool

type EphemeralCredentialTerminalToolService = application.EphemeralCredentialTerminalToolService
type EphemeralCredentialToolService = application.EphemeralCredentialToolService

var ExtractDelegateWorkspace = application.ExtractDelegateWorkspace

type GrokToolService = application.GrokToolService
type HeartbeatToolService = application.HeartbeatToolService

var ImageAnalyzeTool = application.ImageAnalyzeTool
var ImageGenerateTool = application.ImageGenerateTool

type ImageGenerationDefaults = application.ImageGenerationDefaults

var ImageReadTool = application.ImageReadTool
var ImageTool = application.ImageTool
var ListTasksTool = application.ListTasksTool

type MemoryToolService = application.MemoryToolService
type NapCatCrossGroupReadConfig = application.NapCatCrossGroupReadConfig
type NapCatGroupReader = application.NapCatGroupReader

var NapCatReadGroupTool = application.NapCatReadGroupTool
var NewAutonomyToolService = application.NewAutonomyToolService
var NewBuiltinToolService = application.NewBuiltinToolService
var NewCodexToolService = application.NewCodexToolService
var NewComputerUseToolService = application.NewComputerUseToolService
var NewCronToolService = application.NewCronToolService
var NewDelegateManager = application.NewDelegateManager
var NewEphemeralCredentialTerminalToolService = application.NewEphemeralCredentialTerminalToolService
var NewEphemeralCredentialToolService = application.NewEphemeralCredentialToolService
var NewGrokToolService = application.NewGrokToolService
var NewHeartbeatToolService = application.NewHeartbeatToolService
var NewMemoryToolService = application.NewMemoryToolService
var NewPriorityTaskQueue = application.NewPriorityTaskQueue
var NewRAGToolService = application.NewRAGToolService
var NewServices = application.NewServices
var NewTaskCache = application.NewTaskCache

type ParallelDelegateResult = application.ParallelDelegateResult
type ParallelDelegateTask = application.ParallelDelegateTask

var ParseTaskPriority = application.ParseTaskPriority

type PrioritizedTask = application.PrioritizedTask

const PriorityCritical = application.PriorityCritical
const PriorityHigh = application.PriorityHigh
const PriorityLow = application.PriorityLow
const PriorityNormal = application.PriorityNormal

type PriorityTaskQueue = application.PriorityTaskQueue
type RAGToolService = application.RAGToolService

var RegisterBuiltinComputerTools = application.RegisterBuiltinComputerTools
var RegisterBuiltinTools = application.RegisterBuiltinTools
var RegisterBuiltinToolsWithConfig = application.RegisterBuiltinToolsWithConfig

type Services = application.Services

const StatusCancelled = application.StatusCancelled
const StatusCompleted = application.StatusCompleted
const StatusFailed = application.StatusFailed
const StatusPending = application.StatusPending
const StatusRunning = application.StatusRunning

var SubagentTool = application.SubagentTool

type TTSDefaults = application.TTSDefaults

const TargetAgent = application.TargetAgent
const TargetMCP = application.TargetMCP
const TargetSkill = application.TargetSkill

type TaskCache = application.TaskCache
type TaskPriority = application.TaskPriority
type TaskStatus = application.TaskStatus

var TaskStatusTool = application.TaskStatusTool
var TextToSpeechTool = application.TextToSpeechTool
var WaitForTasksTool = application.WaitForTasksTool
