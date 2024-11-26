package main

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"os"
)

func main() {
	// 设置日志级别
	_ = os.Setenv("LOG_LEVEL", "info")
	_ = driverbox.Plugins.RegisterAllPlugins()
	driverbox.Exports.LoadAllExports()
	driverbox.Start()
	select {}
}
