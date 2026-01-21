package main

import (
	"os"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/exports"
	"github.com/ibuilding-x/driver-box/plugins"
)

func main() {
	// 设置日志级别
	_ = os.Setenv("LOG_LEVEL", "info")
	//_ = plugins.EnableAllPlugins()
	plugins.EnableAllPlugins()
	exports.EnableAllExports()
	driverbox.Start()
	select {}
}
