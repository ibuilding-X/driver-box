package driverbox

import (
	"fmt"
	"os"

	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	shadow0 "github.com/ibuilding-x/driver-box/driverbox/shadow"
	"github.com/ibuilding-x/driver-box/internal/cache"
	"github.com/ibuilding-x/driver-box/internal/core"
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"github.com/ibuilding-x/driver-box/internal/shadow"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/crontab"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"go.uber.org/zap"
)

func Start() error {
	//第一步：加载配置文件DriverConfig
	err := initEnvConfig()
	if err != nil {
		fmt.Println("init env config error", err)
		return err
	}

	//第二步：初始化日志记录器
	logger.InitLogger(os.Getenv(config.ENV_LOG_PATH), os.Getenv(config.ENV_LOG_LEVEL))

	//第四步：启动Export
	for _, item := range export0.Exports {
		if err := item.Init(); err != nil {
			Log().Error("init export error", zap.Error(err))
		}
	}

	//第六步：启动driver-box插件
	err = loadPlugins()
	if err != nil {
		Log().Error(err.Error())
	}

	if err != nil {
		TriggerEvents(event.EventCodeServiceStatus, GetMetadata().SerialNo, event.ServiceStatusError)
	} else {
		TriggerEvents(event.EventCodeServiceStatus, GetMetadata().SerialNo, event.ServiceStatusHealthy)
	}

	Log().Info("start driver-box success.")
	return err
}

func Stop() error {
	var e error
	//清理存量定时器
	crontab.Instance().Clear()

	for _, item := range export0.Exports {
		e = item.Destroy()
		if e != nil {
			Log().Error("destroy export error", zap.Error(e))
		}
	}
	export0.Exports = make([]export.Export, 0)
	destroyPlugins()
	plugins.Clear()
	shadow.Reset()
	// 4. 清除核心缓存数据
	cache.Reset()
	return nil
}

func initEnvConfig() error {
	dir := os.Getenv(config.ENV_RESOURCE_PATH)
	if dir == "" {
		config.ResourcePath = "./res"
	} else {
		config.ResourcePath = dir
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

// GetMetadata 获取服务元数据信息
// 返回当前服务的核心元数据配置，包括序列号等关键信息
func GetMetadata() config.Metadata {
	return core.Metadata
}

// CoreCache 获取核心缓存实例
// 提供对系统核心缓存的访问，用于存储和检索运行时数据
func CoreCache() cache.CoreCache {
	return cache.Get()
}

// Shadow 获取设备影子服务实例
// 设备影子服务用于维护设备状态，提供状态同步和监控功能
func Shadow() shadow0.DeviceShadow {
	return shadow.Shadow()
}

func AddFunc(s string, f func()) (*crontab.Future, error) {
	return crontab.Instance().AddFunc(s, f)
}
