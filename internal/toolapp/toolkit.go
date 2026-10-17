// Shared contracts and factories are aliases to their single implementation.
package tool

import toolkit "github.com/yurika0211/luckyagent-tools"

var AskUserTool = toolkit.AskUserTool
var CalculateTool = toolkit.CalculateTool

const CatBuiltin = toolkit.CatBuiltin
const CatDelegate = toolkit.CatDelegate

type Category = toolkit.Category

var CurrentTimeTool = toolkit.CurrentTimeTool
var DataTool = toolkit.DataTool
var DocumentReadTool = toolkit.DocumentReadTool

type ExecutionContext = toolkit.ExecutionContext

var FileDeleteTool = toolkit.FileDeleteTool
var FileListTool = toolkit.FileListTool
var FileMkdirTool = toolkit.FileMkdirTool
var FileMoveTool = toolkit.FileMoveTool
var FilePatchTool = toolkit.FilePatchTool
var FileReadTool = toolkit.FileReadTool
var FileWriteTool = toolkit.FileWriteTool

type FilesystemPolicy = toolkit.FilesystemPolicy

var HTTPRequestTool = toolkit.HTTPRequestTool
var MemoryTool = toolkit.MemoryTool
var NewGateway = toolkit.NewGateway
var NewRegistry = toolkit.NewRegistry

type Observation = toolkit.Observation
type OpenCLIConfig = toolkit.OpenCLIConfig

var OpenCLITool = toolkit.OpenCLITool

type Param = toolkit.Param
type PendingApproval = toolkit.PendingApproval

const PermApprove = toolkit.PermApprove
const PermAuto = toolkit.PermAuto

type PermissionLevel = toolkit.PermissionLevel

var RAGIndexTool = toolkit.RAGIndexTool
var RAGSearchTool = toolkit.RAGSearchTool

type Rect = toolkit.Rect
type Registry = toolkit.Registry

var RequestCredentialTool = toolkit.RequestCredentialTool

type ShellContext = toolkit.ShellContext
type SkillToolService = toolkit.SkillToolService

var TerminalTool = toolkit.TerminalTool

type Tool = toolkit.Tool
type ToolCallResult = toolkit.ToolCallResult

var WebFetchTool = toolkit.WebFetchTool

type WebSearchConfig = toolkit.WebSearchConfig

var WebSearchTool = toolkit.WebSearchTool

var boolArg = toolkit.BoolArg
var boundedIntArg = toolkit.BoundedIntArg
var cloneToolArgs = toolkit.CloneToolArgs
var expandSandboxPath = toolkit.ExpandSandboxPath
var filesystemPolicyFromOptional = toolkit.FilesystemPolicyFromOptional
var mergedToolParams = toolkit.MergeToolParams
var numberArg = toolkit.NumberArg
var paramSchema = toolkit.ParamSchema
var prettyStructuredValue = toolkit.PrettyStructuredValue
var resolveWorkspacePath = toolkit.ResolveWorkspacePath
var sandboxWorkspaceDir = toolkit.SandboxWorkspaceDir
var stringArg = toolkit.StringArg
var toolAction = toolkit.ToolAction
var validateFetchURL = toolkit.ValidateFetchURL
var validateHTTPMethod = toolkit.ValidateHTTPMethod
var validatePath = toolkit.ValidatePath
var validateRequestHeader = toolkit.ValidateRequestHeader
