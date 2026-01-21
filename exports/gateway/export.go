package gateway

import (
	"github.com/ibuilding-x/driver-box/driverbox"
)

// LoadGatewayExport 加载网关Export插件
// 功能:
//
//	创建并加载gwexport.New()实例
func LoadExport() {
	driverbox.RegisterExport(New())
}
