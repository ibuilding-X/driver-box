package mirror

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/exports/mirror/internal"
)

// LoadMirrorExport 加载镜像设备Export插件
// 功能:
//
//	创建并加载mirror.NewExport()实例
func LoadExport() {
	driverbox.RegisterExport(internal.NewExport())
}
