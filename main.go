package main

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/exports"
	"github.com/ibuilding-x/driver-box/plugins"
	"os"
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
