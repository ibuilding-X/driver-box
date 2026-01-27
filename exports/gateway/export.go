package gateway

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/exports/gateway/internal/plugin"
)

// LoadGatewayExport 加载网关Export插件
// 功能:
//
//	创建并加载gwexport.New()实例
func EnableExport() {
	driverbox.EnablePlugin(plugin.ProtocolName, plugin.New())
	driverbox.EnableExport(&gatewayExport{
		wss: &websocketService{},
	})
}
