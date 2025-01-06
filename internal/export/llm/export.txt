package llm

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"os"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

type Export struct {
	llm   llms.Model
	ready bool
}

func (export *Export) Init() error {
	if os.Getenv(config.ENV_EXPORT_LLM_AGENT_ENABLED) == "false" {
		helper.Logger.Warn("driver-box llm-agent is disabled")
		return nil
	}
	//initialization of the llm model (with tinydolphin)
	llm, err := ollama.New(ollama.WithModel("qwen2.5:latest"))
	//llm, err := ollama.New(ollama.WithModel("qwen2:0.5b"))
	if err != nil {
		return err
	}
	export.llm = llm
	export.ready = true
	return nil
}
func NewExport() *Export {
	once.Do(func() {
		driverInstance = &Export{}
	})

	return driverInstance
}

// 点位变化触发场景联动
func (export *Export) ExportTo(deviceData plugin.DeviceData) {
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {

	return nil
}

func (export *Export) IsReady() bool {
	return export.ready
}
