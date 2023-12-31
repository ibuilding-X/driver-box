package driverbox

import (
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/bootstrap"
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/restful/route"
	"go.uber.org/zap"
)

func RegisterPlugin(name string, plugin plugin.Plugin) error {
	return plugins.Manager.Register(name, plugin)
}

func Start(exports []export.Export) {
	//第一步：加载配置文件DriverConfig

	//第二步：初始化日志记录器
	if err := helper.InitLogger("DEBUG"); err != nil {
		return
	}

	//第三步：启动driver-box插件
	if err := bootstrap.LoadPlugins(); err != nil {
		helper.Logger.Error(err.Error())
		return
	}

	// 第四步：启动 REST 服务
	go func() {
		e := route.Register()
		if e != nil {
			helper.Logger.Error("start rest server error", zap.Error(e))
		}
	}()

	// 正式环境需注释掉
	//localMode("192.168.16.88", "59999", "127.0.0.1")
	//第五步：启动Export
	helper.Exports = exports
	for _, item := range exports {
		go func(item export.Export) {
			err := item.Init()
			if err != nil {
				panic(err)
			}
		}(item)
	}

	select {}
}
