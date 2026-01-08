package ui

import (
	"github.com/ibuilding-x/driver-box/internal/export/ui"
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
)

// LoadUIExport 加载driver-box内置UI Export插件
// 功能:
//
//	创建并加载ui.NewExport()实例
func LoadExport() {
	driverbox.Exports.LoadExport(ui.NewExport())
}
