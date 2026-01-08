package basic

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
	"github.com/ibuilding-x/driver-box/pkg/exports/basic/internal"
)

// LoadBasicExport 加载基础Export插件
// 功能:
//
//	创建并加载basic.NewExport()实例
func LoadExport() {
	driverbox.Exports.LoadExport(internal.NewExport())
}
