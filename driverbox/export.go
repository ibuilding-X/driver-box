package driverbox

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/export/discover"
	"github.com/ibuilding-x/driver-box/internal/export/gwexport"
	"github.com/ibuilding-x/driver-box/internal/export/linkedge"
	"github.com/ibuilding-x/driver-box/internal/export/mirror"
	"github.com/ibuilding-x/driver-box/internal/export/ui"
)

var Exports exports

type exports struct {
}

// 加载自定义的Export
func (exports *exports) LoadExport(export2 export.Export) {
	if !exports.exists(export2) {
		export0.Exports = append(export0.Exports, export2)
	}
}

// 批量加载Export
func (exports *exports) LoadExports(export2 []export.Export) {
	for _, e := range export2 {
		exports.LoadExport(e)
	}
}

// 默认加载driver-box内置的所有Export
func (exports *exports) LoadAllExports() {
	exports.LoadLinkEdgeExport()
	exports.LoadMirrorExport()
	exports.LoadUIExport()
	exports.LoadDiscoverExport()
	exports.LoadGatewayExport()
}

// 加载场景联动 Export
func (exports *exports) LoadLinkEdgeExport() {
	exports.LoadExport(linkedge.NewExport())
}

// 加载镜像设备 Export
func (exports *exports) LoadMirrorExport() {
	exports.LoadExport(mirror.NewExport())
}

// 加载设备自动发现Export
func (exports *exports) LoadDiscoverExport() {
	exports.LoadExport(discover.NewExport())
}

// 加载driver-box内置UI Export
func (exports *exports) LoadUIExport() {
	exports.LoadExport(ui.NewExport())
}

func (exports *exports) LoadGatewayExport() {
	exports.LoadExport(gwexport.New())
}

func (exports *exports) exists(exp export.Export) bool {
	for _, e := range export0.Exports {
		if e == exp {
			return true
		}
	}
	return false
}
