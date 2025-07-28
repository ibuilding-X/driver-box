package linkedge

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/export/linkedge"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"os"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

type EnvConfig struct {
	ConfigPath string
}

type Export struct {
	linkEdge  service
	ready     bool
	EnvConfig EnvConfig
}

func (export *Export) Init() error {
	var err error
	export.EnvConfig, err = initLinkEdgeEnvConfig()
	if err != nil {
		helper.Logger.Error(fmt.Sprintf("init linkEdge env config error:%v", err))
		return err
	}

	//启动场景联动服务
	export.linkEdge = service{
		triggerConditions: make(map[string][]linkedge.DevicePointCondition),
		configs:           make(map[string]linkedge.Config),
		schedules:         make(map[string]*cron.Cron),
		envConfig:         export.EnvConfig,
	}
	err = export.linkEdge.NewService()
	if err != nil {
		helper.Logger.Error(fmt.Sprintf("init linkEdge service error:%v", err))
		return err
	}
	export.ready = true
	return nil
}
func (export *Export) Destroy() error {
	export.ready = false
	for key, c := range export.linkEdge.schedules {
		helper.Logger.Info("stop linkEdge cron", zap.String("id", key))
		c.Stop()
	}
	return nil
}
func NewExport() *Export {
	once.Do(func() {
		driverInstance = &Export{
			EnvConfig: EnvConfig{
				ConfigPath: LinkConfigPath,
			},
		}
	})
	return driverInstance
}

// 点位变化触发场景联动
func (export *Export) ExportTo(deviceData plugin.DeviceData) {
	export.linkEdge.devicePointTriggerHandler(deviceData, false)
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	switch eventCode {
	case event.EventCodeLinkEdgeTrigger:
		helper.Logger.Info("trigger linkEdge", zap.String("id", key), zap.Any("result", eventValue))
	case event.EventCodePluginCallback:
		data, ok := eventValue.([]plugin.DeviceData)
		if !ok {
			helper.Logger.Error("plugin callback data error", zap.Any("eventValue", eventValue))
			return nil
		}
		for _, datum := range data {
			export.linkEdge.devicePointTriggerHandler(datum, true)
		}
	}
	return nil
}

func (export *Export) IsReady() bool {
	return export.ready
}

// 初始化场景联动环境配置
func initLinkEdgeEnvConfig() (EnvConfig, error) {
	var envConfig = EnvConfig{}
	//驱动配置文件存放目录
	dir := os.Getenv(config.ENV_LINKEDGE_CONFIG_PATH)
	if dir == "" {
		envConfig.ConfigPath = LinkConfigPath
	} else {
		envConfig.ConfigPath = dir
	}
	return envConfig, nil
}
