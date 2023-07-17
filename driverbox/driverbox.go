package driverbox

import (
	"github.com/ibuilding-x/driver-box/core/contracts"
	"github.com/ibuilding-x/driver-box/core/helper"
	"github.com/ibuilding-x/driver-box/internal/driver/bootstrap"
	"github.com/ibuilding-x/driver-box/internal/driver/plugins"
	"github.com/ibuilding-x/driver-box/internal/driver/plugins/httpserver"
	"github.com/ibuilding-x/driver-box/internal/driver/restful/route"
	"go.uber.org/zap"
)

func RegisterPlugin(name string, plugin contracts.Plugin) error {
	return plugins.Manager.Register("bacnet", &httpserver.Plugin{})
}

func Start(exports []contracts.Export) {
	//第一步：加载配置文件DriverConfig
	helper.Export = &contracts.DefaultExport{}

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
		go func(item contracts.Export) {
			err := item.Init()
			if err != nil {
				panic(err)
			}
		}(item)
	}

}
