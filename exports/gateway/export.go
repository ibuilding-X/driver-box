package gateway

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/internal/export/gwexport"
)

// LoadGatewayExport 加载网关Export插件
// 功能:
//
//	创建并加载gwexport.New()实例
func LoadExport() {
	driverbox.Exports.LoadExport(gwexport.New())
}
