package driverbox

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/internal/bootstrap"
	"github.com/ibuilding-x/driver-box/driverbox/internal/core"
	plugins0 "github.com/ibuilding-x/driver-box/driverbox/internal/plugins"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var srv *http.Server

func Start() error {
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
	helper.Crontab = crontab.Instance()

	//第四步：启动Export
	for _, item := range loadExports {
		if err := item.Init(); err != nil {
			helper.Logger.Error("init export error", zap.Error(err))
		}
	}

	// 第五步：启动 REST 服务
	go func() {
		srv = &http.Server{Addr: ":" + helper.EnvConfig.HttpListen, Handler: restful.HttpRouter}
		e := srv.ListenAndServe()
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
		TriggerEvents(event.EventCodeServiceStatus, GetMetadata().SerialNo, event.ServiceStatusError)
	} else {
		TriggerEvents(event.EventCodeServiceStatus, GetMetadata().SerialNo, event.ServiceStatusHealthy)
	}

	helper.Logger.Info("start driver-box success.")
	return err
}

func Stop() error {
	var e error
	//清理存量定时器
	if helper.Crontab != nil {
		crontab.Instance().Clear()
		helper.Crontab = nil
	}
	if srv != nil {
		e = srv.Shutdown(context.Background())
		srv = nil
		restful.HttpRouter = httprouter.New()
		http.DefaultServeMux = http.NewServeMux()
	}
	for _, item := range loadExports {
		e = item.Destroy()
		if e != nil {
			helper.Logger.Error("destroy export error", zap.Error(e))
		}
	}
	loadExports = make([]export.Export, 0)
	bootstrap.DestroyPlugins()
	plugins0.Manager.Clear()
	// 3. 停止影子服务设备状态监听、删除影子服务
	helper.DeviceShadow.StopStatusListener()
	helper.DeviceShadow = nil
	// 4. 清除核心缓存数据
	helper.CoreCache.Reset()
	helper.CoreCache = nil
	return nil
}

func initEnvConfig() error {
	helper.EnvConfig = config.EnvConfig{}
	dir := os.Getenv(config.ENV_RESOURCE_PATH)
	if dir == "" {
		config.ResourcePath = "./res"
	} else {
		config.ResourcePath = dir
	}
	//驱动配置文件存放目录
	helper.EnvConfig.ConfigPath = path.Join(config.ResourcePath, "driver")
	//http服务绑定host
	httpListen := os.Getenv(config.ENV_HTTP_LISTEN)
	if httpListen != "" {
		helper.EnvConfig.HttpListen = httpListen
	} else {
		helper.EnvConfig.HttpListen = "8081"
	}

	logPath := os.Getenv(config.ENV_LOG_PATH)
	if logPath != "" {
		helper.EnvConfig.LogPath = logPath
	}
	return nil
}

// 触发某个设备点位的读取动作，指令会下发值驱动层
func ReadPoint(deviceId string, pointName string) error {
	return core.SendSinglePoint(deviceId, plugin.ReadMode, plugin.PointData{
		PointName: pointName,
	})
}

// 触发某个设备点位的写入操作
func WritePoint(deviceId string, pointData plugin.PointData) error {
	return core.SendSinglePoint(deviceId, plugin.WriteMode, pointData)
}

// 批量写点位
func WritePoints(deviceId string, pointData []plugin.PointData) error {
	return core.SendBatchWrite(deviceId, pointData)
}

func ReadPoints(deviceId string, pointData []plugin.PointData) error {
	return core.SendBatchRead(deviceId, pointData)
}

//// 获取当前被注册至 driver-box 的所有export
//func GetExports() []export.Export {
//	return export0.Exports
//}

func UpdateMetadata(f func(*config.Metadata)) {
	f(&core.Metadata)
}

func GetMetadata() config.Metadata {
	return core.Metadata
}
