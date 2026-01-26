package driverbox

import (
	"fmt"
	"os"

	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/cache"
	"github.com/ibuilding-x/driver-box/internal/core"
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/export/base"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"github.com/ibuilding-x/driver-box/internal/shadow"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/crontab"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"go.uber.org/zap"
)

// Start 启动driver-box服务
// 该函数执行完整的启动流程，包括：
// 1. 初始化环境配置
// 2. 初始化日志记录器
// 3. 启动所有Export模块
// 4. 启动所有插件
// 5. 触发服务状态事件
// 返回值:
//
//	error: 启动过程中发生的任何错误，如配置加载失败、端口占用等
//
// 使用示例:
//
//	if err := driverbox.Start(); err != nil {
//	    driverbox.Log().Fatal("Failed to start driver-box", zap.Error(err))
//	}
func Start() error {
	// 第一步：加载配置文件DriverConfig
	err := initEnvConfig()
	if err != nil {
		fmt.Println("init env config error", err)
		return err
	}

	// 第二步：初始化日志记录器
	logger.InitLogger(os.Getenv(config.ENV_LOG_PATH), os.Getenv(config.ENV_LOG_LEVEL))

	// 第三步：启动Export
	for _, item := range export0.Exports {
		if err := item.Init(); err != nil {
			Log().Error("init export error", zap.Error(err))
		}
	}

	// 第四步：启动driver-box插件
	err = loadPlugins()
	if err != nil {
		Log().Error(err.Error())
	}

	if err != nil {
		TriggerEvents(event.ServiceStatus, GetMetadata().SerialNo, event.ServiceStatusError)
		Log().Info("start driver-box error.")
		_ = Stop()
	} else {
		TriggerEvents(event.ServiceStatus, GetMetadata().SerialNo, event.ServiceStatusHealthy)
		Log().Info("start driver-box success.")
	}

	return err
}

// Stop 停止driver-box服务
// 该函数执行完整的停止流程，包括：
// 1. 清理所有定时器任务
// 2. 销毁所有Export模块
// 3. 销毁所有插件
// 4. 重置影子服务
// 5. 清除核心缓存数据
// 返回值:
//
//	error: 停止过程中发生的任何错误(当前总是返回nil)
//
// 使用示例:
//
//	if err := driverbox.Stop(); err != nil {
//	    driverbox.Log().Error("Error stopping driver-box", zap.Error(err))
//	}
func Stop() error {
	var e error
	// 第一步：清理存量定时器
	crontab.Instance().Clear()

	// 第二步：销毁所有Export模块
	for _, item := range export0.Exports {
		e = item.Destroy()
		if e != nil {
			Log().Error("destroy export error", zap.Error(e))
		}
	}
	export0.Exports = make([]export.Export, 0)
	// 重新注册基础Export
	EnableExport(base.Get())

	// 第三步：销毁插件
	destroyPlugins()
	plugins.clear()

	// 第四步：重置影子服务
	shadow.Reset()

	// 第五步：清除核心缓存数据
	cache.Reset()

	return nil
}

// initEnvConfig 初始化环境配置
// 设置资源配置路径，默认为"./res"
// 返回值:
//
//	error: 初始化过程中发生的错误，当前实现总是返回nil
func initEnvConfig() error {
	dir := os.Getenv(config.ENV_RESOURCE_PATH)
	if dir == "" {
		config.ResourcePath = "./res"
	} else {
		config.ResourcePath = dir
	}

	return nil
}

// ReadPoint 触发对指定设备点位的读取操作
// 该函数会将读取指令下发到驱动层
// 参数:
//
//	deviceId: 设备唯一标识符
//	pointName: 需要读取的点位名称
//
// 返回值:
//
//	error: 操作过程中发生的错误，如设备不存在、点位不存在、通信失败等
//
// 使用示例:
//
//	err := driverbox.ReadPoint("device001", "temperature")
//	if err != nil {
//	    driverbox.Log().Error("Read point failed", zap.Error(err))
//	}
func ReadPoint(deviceId string, pointName string) error {
	return core.SendSinglePoint(deviceId, plugin.ReadMode, plugin.PointData{
		PointName: pointName,
	})
}

// WritePoint 触发对指定设备点位的写入操作
// 该函数会将写入指令下发到驱动层
// 参数:
//
//	deviceId: 设备唯一标识符
//	pointData: 包含点位名称和值的结构体
//
// 返回值:
//
//	error: 操作过程中发生的错误，如设备不存在、点位不存在、值类型错误、通信失败等
//
// 使用示例:
//
//	data := plugin.PointData{
//	    PointName: "switch",
//	    Value:     true,
//	}
//	err := driverbox.WritePoint("device001", data)
//	if err != nil {
//	    driverbox.Log().Error("Write point failed", zap.Error(err))
//	}
func WritePoint(deviceId string, pointData plugin.PointData) error {
	return core.SendSinglePoint(deviceId, plugin.WriteMode, pointData)
}

// WritePoints 批量写入多个设备点位
// 该函数会将批量写入指令下发到驱动层
// 参数:
//
//	deviceId: 设备唯一标识符
//	pointData: 包含多个点位名称和值的数组
//
// 返回值:
//
//	error: 操作过程中发生的错误，如设备不存在、部分点位不存在、值类型错误、通信失败等
//
// 使用示例:
//
//	points := []plugin.PointData{
//	    {PointName: "switch1", Value: true},
//	    {PointName: "switch2", Value: false},
//	}
//	err := driverbox.WritePoints("device001", points)
//	if err != nil {
//	    driverbox.Log().Error("Batch write points failed", zap.Error(err))
//	}
func WritePoints(deviceId string, pointData []plugin.PointData) error {
	return core.SendBatchWrite(deviceId, pointData)
}

// ReadPoints 批量读取多个设备点位
// 该函数会将批量读取指令下发到驱动层
// 参数:
//
//	deviceId: 设备唯一标识符
//	pointData: 包含多个点位名称的数组
//
// 返回值:
//
//	error: 操作过程中发生的错误，如设备不存在、部分点位不存在、通信失败等
//
// 使用示例:
//
//	points := []plugin.PointData{
//	    {PointName: "temperature"},
//	    {PointName: "humidity"},
//	}
//	err := driverbox.ReadPoints("device001", points)
//	if err != nil {
//	    driverbox.Log().Error("Batch read points failed", zap.Error(err))
//	}
func ReadPoints(deviceId string, pointData []plugin.PointData) error {
	return core.SendBatchRead(deviceId, pointData)
}
