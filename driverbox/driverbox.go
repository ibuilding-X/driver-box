package driverbox

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/bootstrap"
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/restful/route"
	"go.uber.org/zap"
	"os"
)

// 网关编号
var SerialNo = "driver-box"

func RegisterPlugin(name string, plugin plugin.Plugin) error {
	return plugins.Manager.Register(name, plugin)
}

func Start(exports []export.Export) error {
	//第一步：加载配置文件DriverConfig
	err := initEnvConfig()
	if err != nil {
		fmt.Println("init env config error", err)
		return err
	}

	//第二步：初始化日志记录器
	if err := helper.InitLogger(os.Getenv("LOG_LEVEL")); err != nil {
		fmt.Println("init logger error", err)
		return err
	}
	//第三步：启动定时器
	helper.Crontab = crontab.NewCrontab()
	helper.Crontab.Start()

	//第四步：启动Export
	helper.Exports = exports
	for _, item := range exports {
		if err := item.Init(); err != nil {
			helper.Logger.Error("init export error", zap.Error(err))
		}
	}

	// 第五步：启动 REST 服务
	go func() {
		e := route.Register()
		if e != nil {
			helper.Logger.Error("start rest server error", zap.Error(e))
		}
	}()

	//第六步：启动driver-box插件
	err = bootstrap.LoadPlugins()
	if err != nil {
		helper.Logger.Error(err.Error())
	}

	if err != nil {
		helper.TriggerEvents(event.EventCodeServiceStatus, SerialNo, event.ServiceStatusError)
	} else {
		helper.TriggerEvents(event.EventCodeServiceStatus, SerialNo, event.ServiceStatusHealthy)
	}

	helper.Logger.Info("start driver-box success.")
	return err
}

func initEnvConfig() error {
	helper.EnvConfig = config.EnvConfig{}
	//驱动配置文件存放目录
	dir := os.Getenv(config.ENV_CONFIG_PATH)
	if dir == "" {
		helper.EnvConfig.ConfigPath = "./driver-config"
	} else {
		helper.EnvConfig.ConfigPath = dir
	}
	//http服务绑定host
	httpListen := os.Getenv(config.ENV_HTTP_LISTEN)
	if httpListen != "" {
		helper.EnvConfig.HttpListen = httpListen
	} else {
		helper.EnvConfig.HttpListen = ":8081"
	}

	logPath := os.Getenv(config.ENV_LOG_PATH)
	if logPath != "" {
		helper.EnvConfig.LogPath = logPath
	}
	return nil
}
