package gateway

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type gatewayExport struct {
	wss *websocketService
}

// Init 初始化
func (g *gatewayExport) Init() error {
	g.wss.Start()
	return nil
}

func (g *gatewayExport) Destroy() error {
	return nil
}

// ExportTo 接收驱动数据
func (g *gatewayExport) ExportTo(deviceData plugin.DeviceData) {
	g.wss.sendDeviceData(deviceData)
}

// OnEvent 接收事件数据
func (g *gatewayExport) OnEvent(eventCode string, key string, eventValue interface{}) error {
	// 暂时不处理任何事件
	return nil
}

func (g *gatewayExport) IsReady() bool {
	return true
}

func New() export.Export {
	return &gatewayExport{
		wss: &websocketService{},
	}
}
