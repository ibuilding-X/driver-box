package manager

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"sync"
)

var driverInstance *Export
var once = &sync.Once{}

// 设备自动发现插件
type Export struct {
	ready bool
}

func (export *Export) Init() error {
	export.ready = true
	registerApi()
	udpDiscover()
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
