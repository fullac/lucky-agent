package tool

import (
	toolkit "github.com/yurika0211/luckyagent-tools"
	"github.com/yurika0211/luckyagent/internal/multimodal"
)

// RegisterBuiltinTools 注册所有内置工具
func RegisterBuiltinTools(r *Registry, mediaProcessor ...*multimodal.Processor) {
	RegisterBuiltinToolsWithConfig(r, nil, nil, mediaProcessor...)
}

// RegisterBuiltinToolsWithConfig 注册所有内置工具（带搜索配置）
func RegisterBuiltinToolsWithConfig(r *Registry, searchCfg *WebSearchConfig, opencliCfg *OpenCLIConfig, mediaProcessor ...*multimodal.Processor) {
	var processor *multimodal.Processor
	if len(mediaProcessor) > 0 {
		processor = mediaProcessor[0]
	}

	toolkit.RegisterBuiltinToolsWithConfig(r, searchCfg, opencliCfg)
	r.Register(ImageTool(processor, "", nil, ImageGenerationDefaults{}))
	r.Register(TextToSpeechTool(nil, TTSDefaults{}))
}

// RegisterBuiltinComputerTools adds the optional desktop observation/control
// tools to an existing registry. Keeping this separate avoids constructing a
// platform backend when computer use is disabled.
func RegisterBuiltinComputerTools(r *Registry, manager ComputerManager, cfg ComputerUseConfig) {
	if r == nil || manager == nil {
		return
	}
	NewComputerUseToolService(manager, cfg).RegisterTools(r)
}
