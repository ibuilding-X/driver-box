package mirror

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/plugins/mirror"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

type Export struct {
	ready  bool
	plugin *mirror.Plugin
}

func (export *Export) Init() error {
	//注册镜像插件
	export.plugin = new(mirror.Plugin)
	driverbox.RegisterPlugin("mirror", export.plugin)
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
	callback.OnReceiveHandler(export.plugin.VirtualConnector, deviceData)
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	return nil
}

func (export *Export) IsReady() bool {
	return export.ready
}
