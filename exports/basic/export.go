package basic

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/exports/basic/internal"
)

// LoadBasicExport 加载基础Export插件
// 功能:
//
//	创建并加载basic.NewExport()实例
func EnableExport() {
	driverbox.RegisterExport(internal.NewExport())
}

func Get() internal.Api {
	return internal.NewExport()
}
