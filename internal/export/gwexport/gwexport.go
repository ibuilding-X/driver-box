package gwexport

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type gatewayExport struct {
}

func (g *gatewayExport) Init() error {
	//TODO implement me
	panic("implement me")
}

func (g *gatewayExport) ExportTo(deviceData plugin.DeviceData) {
	//TODO implement me
	panic("implement me")
}

func (g *gatewayExport) OnEvent(eventCode string, key string, eventValue interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (g *gatewayExport) IsReady() bool {
	//TODO implement me
	panic("implement me")
}

func New() export.Export {
	return &gatewayExport{}
}
