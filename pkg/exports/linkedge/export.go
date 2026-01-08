package linkedge

import (
	"github.com/ibuilding-x/driver-box/internal/export/linkedge"
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
)

// LoadLinkEdgeExport 加载场景联动Export插件
// 功能:
//
//	创建并加载linkedge.NewExport()实例
func LoadExport() {
	driverbox.Exports.LoadExport(linkedge.NewExport())
}
