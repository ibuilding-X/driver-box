package main

import (
	"os"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/exports"
	"github.com/ibuilding-x/driver-box/v2/plugins"
)

func main() {
	// 设置日志级别
	_ = os.Setenv("LOG_LEVEL", "info")
	plugins.EnableAll()
	exports.EnableAll()
	driverbox.Start()
	select {}
}
