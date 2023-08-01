package export

import "github.com/ibuilding-x/driver-box/driverbox/plugin"

type Export interface {
	Init() error
	// ExportTo 导出消息：写入Edgex总线、MQTT上云
	ExportTo(deviceData plugin.DeviceData)

	SendStatusChangeNotification(deviceName string, online bool) error

	IsReady() bool
}
