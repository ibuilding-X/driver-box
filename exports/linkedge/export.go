package linkedge

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/exports/linkedge/internal"
)

// LoadLinkEdgeExport 加载场景联动Export插件
// 功能:
//
//	创建并加载linkedge.NewExport()实例
func EnableExport() {
	driverbox.EnableExport(internal.NewExport())
}
