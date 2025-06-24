package driverbox

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/export/basic"
	"github.com/ibuilding-x/driver-box/internal/export/discover"
	"github.com/ibuilding-x/driver-box/internal/export/gwexport"
	"github.com/ibuilding-x/driver-box/internal/export/linkedge"
	"github.com/ibuilding-x/driver-box/internal/export/mcp"
	"github.com/ibuilding-x/driver-box/internal/export/mirror"
	"github.com/ibuilding-x/driver-box/internal/export/ui"
)

var Exports exports

// exports 结构体用于管理driver-box框架中的所有Export插件
// 提供加载单个、批量加载以及加载所有内置Export的方法
type exports struct {
}

// LoadExport 加载单个自定义Export插件
// 参数:
//
//	export2: 需要加载的Export插件实例
//
// 功能:
//
//	如果该Export尚未加载，则将其添加到全局Exports列表中
func (exports *exports) LoadExport(export2 export.Export) {
	if !exports.exists(export2) {
		export0.Exports = append(export0.Exports, export2)
	}
}

// LoadExports 批量加载多个Export插件
// 参数:
//
//	export2: 需要加载的Export插件实例数组
//
// 功能:
//
//	遍历数组并调用LoadExport方法逐个加载
func (exports *exports) LoadExports(export2 []export.Export) {
	for _, e := range export2 {
		exports.LoadExport(e)
	}
}

// LoadAllExports 加载driver-box框架内置的所有Export插件
// 功能:
//
//	依次调用各个内置Export的加载方法，包括基础Export、场景联动Export等
func (exports *exports) LoadAllExports() {
	exports.LoadBasicExport()
	exports.LoadLinkEdgeExport()
	exports.LoadMirrorExport()
	exports.LoadUIExport()
	exports.LoadDiscoverExport()
	exports.LoadGatewayExport()
}

// LoadLinkEdgeExport 加载场景联动Export插件
// 功能:
//
//	创建并加载linkedge.NewExport()实例
func (exports *exports) LoadLinkEdgeExport() {
	exports.LoadExport(linkedge.NewExport())
}

// LoadMirrorExport 加载镜像设备Export插件
// 功能:
//
//	创建并加载mirror.NewExport()实例
func (exports *exports) LoadMirrorExport() {
	exports.LoadExport(mirror.NewExport())
}

// LoadDiscoverExport 加载设备自动发现Export插件
// 功能:
//
//	创建并加载discover.NewExport()实例
func (exports *exports) LoadDiscoverExport() {
	exports.LoadExport(discover.NewExport())
}

// LoadUIExport 加载driver-box内置UI Export插件
// 功能:
//
//	创建并加载ui.NewExport()实例
func (exports *exports) LoadUIExport() {
	exports.LoadExport(ui.NewExport())
}

// LoadGatewayExport 加载网关Export插件
// 功能:
//
//	创建并加载gwexport.New()实例
func (exports *exports) LoadGatewayExport() {
	exports.LoadExport(gwexport.New())
}

// LoadBasicExport 加载基础Export插件
// 功能:
//
//	创建并加载basic.NewExport()实例
func (exports *exports) LoadBasicExport() {
	exports.LoadExport(basic.NewExport())
}

func (exports *exports) LoadMcpExport() {
	exports.LoadExport(mcp.NewExport())
}

// exists 检查指定的Export是否已经加载
// 参数:
//
//	exp: 需要检查的Export实例
//
// 返回值:
//
//	bool: true表示已加载，false表示未加载
func (exports *exports) exists(exp export.Export) bool {
	for _, e := range export0.Exports {
		if e == exp {
			return true
		}
	}
	return false
}
