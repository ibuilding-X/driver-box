package driverbox

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/export/discover"
	"github.com/ibuilding-x/driver-box/internal/export/linkedge"
	"github.com/ibuilding-x/driver-box/internal/export/mirror"
	"github.com/ibuilding-x/driver-box/internal/export/ui"
)

var Exports exports

type exports struct {
}

// 默认加载driver-box内置的所有Export
func (exports *exports) LoadAllExports() {
	exports.LoadLinkEdgeExport()
	exports.LoadMirrorExport()
	exports.LoadUIExport()
	exports.LoadDiscoverExport()
}

// 加载场景联动 Export
func (exports *exports) LoadLinkEdgeExport() {
	if exp := linkedge.NewExport(); !exports.exists(exp) {
		export0.Exports = append(export0.Exports, exp)
	}
}

// 加载镜像设备 Export
func (exports *exports) LoadMirrorExport() {
	if exp := mirror.NewExport(); !exports.exists(exp) {
		export0.Exports = append(export0.Exports, exp)
	}
}

// 加载设备自动发现Export
func (exports *exports) LoadDiscoverExport() {
	if exp := discover.NewExport(); !exports.exists(exp) {
		export0.Exports = append(export0.Exports, exp)
	}
}

// 加载driver-box内置UI Export
func (exports *exports) LoadUIExport() {
	if exp := ui.NewExport(); !exports.exists(exp) {
		export0.Exports = append(export0.Exports, exp)
	}
}
func (exports *exports) exists(exp export.Export) bool {
	for _, e := range export0.Exports {
		if e == exp {
			return true
		}
	}
	return false
}
