package discover

import (
	"github.com/ibuilding-x/driver-box/internal/export/discover"
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
)

// LoadDiscoverExport 加载设备自动发现Export插件
// 功能:
//
//	创建并加载discover.NewExport()实例
func LoadExport() {
	driverbox.Exports.LoadExport(discover.NewExport())
}
