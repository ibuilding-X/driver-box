package internal

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/exports/linkedge/model"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var driverInstance *Export
var once = &sync.Once{}

type EnvConfig struct {
	ConfigPath string
}

type Export struct {
	Service   service
	ready     bool
	EnvConfig EnvConfig
}

func (export *Export) Init() error {
	var err error
	export.EnvConfig, err = initLinkEdgeEnvConfig()
	if err != nil {
		driverbox.Log().Error(fmt.Sprintf("init linkEdge env config error:%v", err))
		return err
	}

	//启动场景联动服务
	export.Service = service{
		triggerConditions: make(map[string][]model.DevicePointCondition),
		configs:           make(map[string]model.Config),
		schedules:         make(map[string]*cron.Cron),
		envConfig:         export.EnvConfig,
	}
	err = export.Service.NewService()
	if err != nil {
		driverbox.Log().Error(fmt.Sprintf("init linkEdge service error:%v", err))
		return err
	}
	export.ready = true
	return nil
}
func (export *Export) Destroy() error {
	export.ready = false
	for key, c := range export.Service.schedules {
		driverbox.Log().Info("stop linkEdge cron", zap.String("id", key))
		c.Stop()
	}
	return nil
}
func NewExport() *Export {
	once.Do(func() {
		driverInstance = &Export{
			EnvConfig: EnvConfig{},
		}
	})
	return driverInstance
}

// 点位变化触发场景联动
func (export *Export) ExportTo(deviceData plugin.DeviceData) {
	export.Service.devicePointTriggerHandler(deviceData, false)
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	switch eventCode {
	case event.EventCodeLinkEdgeTrigger:
		driverbox.Log().Info("trigger linkEdge", zap.String("id", key), zap.Any("result", eventValue))
	case event.EventCodePluginCallback:
		data, ok := eventValue.([]plugin.DeviceData)
		if !ok {
			driverbox.Log().Error("plugin callback data error", zap.Any("eventValue", eventValue))
			return nil
		}
		for _, datum := range data {
			export.Service.devicePointTriggerHandler(datum, true)
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
	envConfig.ConfigPath = filepath.Join(config.ResourcePath, "linkedge")
	return envConfig, nil
}
