package main

import (
	"os"

	"github.com/ibuilding-x/driver-box/pkg/driverbox"
	"github.com/ibuilding-x/driver-box/pkg/exports"
	"github.com/ibuilding-x/driver-box/pkg/plugins"
)

func main() {
	// 设置日志级别
	_ = os.Setenv("LOG_LEVEL", "info")
	//_ = plugins.RegisterAllPlugins()
	plugins.RegisterAllPlugins()
	exports.LoadAllExports()
	driverbox.Start()
	select {}
}
