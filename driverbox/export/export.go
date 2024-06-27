package export

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type Export interface {
	Init() error
	// ExportTo 导出消息：写入Edgex总线、MQTT上云
	ExportTo(deviceData plugin.DeviceData)

	//事件触发回调
	OnEvent(eventCode string, key string, eventValue interface{}) error

	IsReady() bool
}
