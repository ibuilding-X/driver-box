package exports

import (
	"github.com/ibuilding-x/driver-box/exports/basic"
	"github.com/ibuilding-x/driver-box/exports/discover"
	"github.com/ibuilding-x/driver-box/exports/gateway"
	"github.com/ibuilding-x/driver-box/exports/linkedge"
	"github.com/ibuilding-x/driver-box/exports/mirror"
	"github.com/ibuilding-x/driver-box/exports/ui"
)

// LoadAllExports 加载driver-box框架内置的所有Export插件
// 功能:
//
//	依次调用各个内置Export的加载方法，包括基础Export、场景联动Export等
func LoadAllExports() {
	basic.LoadExport()
	linkedge.LoadExport()
	mirror.LoadExport()
	ui.LoadExport()
	discover.LoadExport()
	gateway.LoadExport()
}
