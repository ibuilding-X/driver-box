package basic

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/internal/export/basic"
)

// LoadBasicExport 加载基础Export插件
// 功能:
//
//	创建并加载basic.NewExport()实例
func LoadExport() {
	driverbox.Exports.LoadExport(basic.NewExport())
}
