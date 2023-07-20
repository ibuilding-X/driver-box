package export

import "github.com/ibuilding-x/driver-box/driverbox/plugin"

type ExportConfig struct {
}
type Export interface {
	Init() error
	//导出消息：写入Edgex总线、MQTT上云
	ExportTo(deviceData plugin.DeviceData)

	SendStatusChangeNotification(deviceName string, online bool) error

	IsReady() bool
}

type DefaultExport struct {
	init bool
}

func (export *DefaultExport) Init() error {
	export.init = true
	return nil
}

// 导出消息：写入Edgex总线、MQTT上云
func (export *DefaultExport) ExportTo(deviceData plugin.DeviceData) {

}

func (export *DefaultExport) SendStatusChangeNotification(deviceName string, online bool) error {
	return nil
}

func (export *DefaultExport) IsReady() bool {
	return export.init
}
