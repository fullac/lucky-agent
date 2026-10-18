package agent

import "github.com/yurika0211/luckyagent-runtime/bootstrap"

// Compatibility bindings for configuration reload and existing core tests.
// Backend construction and application defaults are implemented in runtime.
type embedderRuntimeConfig = bootstrap.EmbedderRuntimeConfig
type soulRuntime = bootstrap.SoulRuntime
type providerRuntime = bootstrap.ProviderRuntime
type ragRuntime = bootstrap.RagRuntime
type supportRuntime = bootstrap.SupportRuntime
type imageGenerationRuntimeConfig = bootstrap.ImageGenerationRuntimeConfig
type ttsRuntimeConfig = bootstrap.TtsRuntimeConfig
type mediaRuntime = bootstrap.MediaRuntime

var resolveEmbedderRuntimeConfig = bootstrap.ResolveEmbedderRuntimeConfig
var toProviderConfig = bootstrap.ToProviderConfig
var wrapProviderWithMiddleware = bootstrap.WrapProviderWithMiddleware
var retryPredicateFromConfig = bootstrap.RetryPredicateFromConfig
var initSoulRuntime = bootstrap.InitSoulRuntime
var initProviderRuntime = bootstrap.InitProviderRuntime
var buildConfiguredProvider = bootstrap.BuildConfiguredProvider
var initProactiveRuntime = bootstrap.InitProactiveRuntime
var proactiveStorePath = bootstrap.ProactiveStorePath
var initProactiveRuntimeService = bootstrap.InitProactiveRuntimeService
var proactiveAllowedActionMap = bootstrap.ProactiveAllowedActionMap
var initRAGRuntime = bootstrap.InitRAGRuntime
var initSupportRuntime = bootstrap.InitSupportRuntime
var reservedTokensFor = bootstrap.ReservedTokensFor
var buildAutonomyRuntimeConfig = bootstrap.BuildAutonomyRuntimeConfig
var buildDelegateRuntimeConfig = bootstrap.BuildDelegateRuntimeConfig
var buildHookRuntimeConfig = bootstrap.BuildHookRuntimeConfig
var toHookSpecs = bootstrap.ToHookSpecs
var resolveImageGenerationConfig = bootstrap.ResolveImageGenerationConfig
var resolveTTSConfig = bootstrap.ResolveTTSConfig
var applyWebSearchEnv = bootstrap.ApplyWebSearchEnv
var applyOpenCLIEnv = bootstrap.ApplyOpenCLIEnv
var buildMediaRuntime = bootstrap.BuildMediaRuntime
var resolveMemoryRuntime = bootstrap.ResolveMemoryRuntime
var resolveConfiguredCredentials = bootstrap.ResolveConfiguredCredentials
var applyResolvedModelCredential = bootstrap.ApplyResolvedModelCredential
