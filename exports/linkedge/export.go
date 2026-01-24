package linkedge

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

var driverInstance *export
var once = &sync.Once{}

// LoadLinkEdgeExport 加载场景联动Export插件
// 功能:
//
//	创建并加载linkedge.NewExport()实例
func EnableExport() {
	driverbox.EnableExport(newExport())
}

func Service() IService {
	return newExport()
}

type export struct {
	// 场景联动配置缓存
	configs map[string]model.Config
	// 定时任务
	schedules map[string]*cron.Cron
	//点位触发器
	triggerConditions map[string][]model.DevicePointCondition
	ready             bool
	ConfigPath        string
}

func (export *export) Init() error {
	var err error
	export.ConfigPath = filepath.Join(config.ResourcePath, "linkedge")
	//启动场景联动服务
	export.triggerConditions = make(map[string][]model.DevicePointCondition)
	export.configs = make(map[string]model.Config)
	export.schedules = make(map[string]*cron.Cron)
	//启动场景联动
	configs, e := export.GetList()
	if e != nil {
		return e
	}
	for _, config := range configs {
		e = export.registerTrigger(config.ID)
		if e != nil {
			return e
		}
	}

	err = export.NewService()
	if err != nil {
		driverbox.Log().Error(fmt.Sprintf("init linkEdge service error:%v", err))
		return err
	}
	export.ready = true
	return nil
}
func (export *export) Destroy() error {
	export.ready = false
	for key, c := range export.schedules {
		driverbox.Log().Info("stop linkEdge cron", zap.String("id", key))
		c.Stop()
	}
	return nil
}
func newExport() *export {
	once.Do(func() {
		driverInstance = &export{}
	})
	return driverInstance
}

// 点位变化触发场景联动
func (export *export) ExportTo(deviceData plugin.DeviceData) {
	export.devicePointTriggerHandler(deviceData, false)
}

// 继承Export OnEvent接口
func (export *export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	switch eventCode {
	case EVT_Trigger:
		driverbox.Log().Info("trigger linkEdge", zap.String("id", key), zap.Any("result", eventValue))
	case event.EventCodePluginCallback:
		data, ok := eventValue.([]plugin.DeviceData)
		if !ok {
			driverbox.Log().Error("plugin callback data error", zap.Any("eventValue", eventValue))
			return nil
		}
		for _, datum := range data {
			export.devicePointTriggerHandler(datum, true)
		}
	}
	return nil
}

func (export *export) IsReady() bool {
	return export.ready
}
