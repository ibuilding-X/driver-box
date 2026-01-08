package driverbox

import (
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/export"
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
