package driver

import (
	coreConfig "driver-box/core/config"
	"driver-box/core/contracts"
	"driver-box/core/helper"
	"driver-box/core/helper/crontab"
	"driver-box/core/helper/shadow"
	"driver-box/driver/common"
	"driver-box/driver/device"
	"driver-box/driver/plugins"
	"driver-box/driver/restful/route"
	"errors"
	"fmt"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"go.uber.org/zap"
	"time"
)

// initialize 额外初始化工作
func (s *Driver) initialize() error {
	// 初始化日志记录器
	if err := helper.InitLogger(s.serviceConfig.DriverConfig.LoggerLevel); err != nil {
		return common.InitLoggerErr
	}

	// 点位缓存客户端
	helper.InitPointCache(time.Duration(s.serviceConfig.DriverConfig.PointCacheTTL) * time.Second)

	// 加载核心配置
	configMap, err := coreConfig.ParseFromPath(common.CoreConfigPath)
	if err != nil {
		return errors.New(common.LoadCoreConfigErr.Error() + ":" + err.Error())
	}

	// 核心配置校验
	for key, _ := range configMap {
		if err = configMap[key].Validate(); err != nil {
			return fmt.Errorf("[%s] config is error: %s", key, err.Error())
		}
	}

	// 缓存核心配置
	if err = helper.InitCoreCache(configMap); err != nil {
		helper.Logger.Error("init core cache error")
		return err
	}

	// 初始化设备模型及设备
	if err = device.Init(); err != nil {
		return err
	}

	// 初始化本地影子服务
	initDeviceShadow(s.serviceConfig.DriverConfig.DefaultDeviceTTL, configMap)

	// 初始化通知服务
	helper.InitNotification()

	// 注册 REST API 路由
	route.Register()

	// 启动插件
	for key, _ := range configMap {
		helper.Logger.Info(key+" begin start", zap.Any("directoryName", key), zap.Any("plugin", configMap[key].ProtocolName))
		// 获取插件示例
		plugin, err := plugins.Manager.Get(configMap[key])
		if err != nil {
			helper.Logger.Error(err.Error())
			continue
		}

		ls, err := helper.InitLuaVM(key)
		if err != nil {
			helper.Logger.Error(err.Error())
			continue
		}

		err = plugin.Initialize(helper.Logger, configMap[key], receiveHandler(), ls)
		if err != nil {
			return err
		}

		// 缓存插件
		helper.CoreCache.AddRunningPlugin(key, plugin)

		// 初始化定时任务
		if configMap[key].Tasks != nil && len(configMap[key].Tasks) > 0 {
			if err = initTimerTasks(configMap[key].Tasks); err != nil {
				return err
			}
		}

		helper.Logger.Info("start success", zap.Any("directoryName", key), zap.Any("plugin", configMap[key].ProtocolName))
	}

	return nil
}

// receiveHandler 接收消息回调
func receiveHandler() contracts.OnReceiveHandler {
	return func(plugin contracts.Plugin, raw interface{}) (result interface{}, err error) {
		helper.Logger.Debug("raw data", zap.Any("data", raw))
		// 协议适配器
		deviceData, err := plugin.ProtocolAdapter().Decode(raw)
		helper.Logger.Debug("decode data", zap.Any("data", deviceData))
		if err != nil {
			return nil, err
		}
		// 写入消息总线
		for _, data := range deviceData {
			var values []*sdkModels.CommandValue
			for _, point := range data.Values {
				// 获取点位信息
				cachePoint, ok := helper.CoreCache.GetPointByDevice(data.DeviceName, point.PointName)
				if !ok {
					helper.Logger.Warn("unknown point", zap.Any("deviceName", data.DeviceName), zap.Any("pointName", point.PointName))
					continue
				}
				// 缓存比较
				shadowValue, _ := helper.DeviceShadow.GetDevicePoint(data.DeviceName, point.PointName)
				if shadowValue == point.Value {
					helper.Logger.Debug("point value = cache, stop sending to messageBus")
					continue
				}
				// 缓存
				if err = helper.DeviceShadow.SetDevicePoint(data.DeviceName, point.PointName, point.Value); err != nil {
					helper.Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceName", data.DeviceName))
				}
				// 点位类型转换
				pointValue, err := helper.ConvPointType(point.Value, cachePoint.ValueType)
				if err != nil {
					helper.Logger.Warn("point value type convert error", zap.Error(err))
					continue
				}
				// 点位值类型名称转换
				pointType := helper.PointValueType2EdgeX(cachePoint.ValueType)
				v, err := sdkModels.NewCommandValue(point.PointName, pointType, pointValue)
				if err != nil {
					helper.Logger.Warn("new command value error", zap.Error(err), zap.Any("pointName", point.PointName), zap.Any("type", pointType), zap.Any("value", pointValue))
					continue
				}
				values = append(values, v)
			}
			if len(values) > 0 {
				helper.Logger.Info("send to message bus", zap.Any("deviceName", data.DeviceName), zap.Any("values", values))
				helper.MessageBus <- &sdkModels.AsyncValues{
					DeviceName:    data.DeviceName,
					SourceName:    "default",
					CommandValues: values,
				}
			}
		}
		return
	}
}

// 初始化设备 autoEvent
func initTimerTasks(tasks []coreConfig.TimerTask) (err error) {
	c := crontab.NewCrontab()
	for _, task := range tasks {
		// 读点位
		if task.Type == "read_points" {
			for _, action := range task.Action {
				if len(action.DeviceNames) > 0 && len(action.Points) > 0 {
					if err := c.AddFunc(task.Interval+"ms", timerTaskHandler(action.DeviceNames, action.Points)); err != nil {
						return err
					}
				}
			}
		}
	}

	c.Start()
	return
}

// timerTaskHandler autoEvent处理函数
func timerTaskHandler(deviceNames []string, pointNames []string) func() {
	return func() {
		helper.Logger.Info("begin handle auto event",
			zap.String("deviceNames", fmt.Sprintf("%+v", deviceNames)),
			zap.String("pointNames", fmt.Sprintf("%+v", pointNames)))
		if err := helper.SendMultiRead(deviceNames, pointNames); err != nil {
			helper.Logger.Error("auto event send error", zap.Error(err))
		}
	}
}

// 初始化影子服务
func initDeviceShadow(defaultTTL int64, configMap map[string]coreConfig.Config) {
	shadow.SetDefaultDeviceTTL(time.Duration(defaultTTL) * time.Second)
	// 设置影子服务设备生命周期
	helper.DeviceShadow = shadow.NewDeviceShadow()
	// 设置回调
	helper.DeviceShadow.SetOnlineChangeCallback(func(deviceName string, online bool) {
		if online {
			helper.Logger.Info("device online", zap.String("deviceName", deviceName))
		} else {
			helper.Logger.Warn("device offline...", zap.String("deviceName", deviceName))
		}
		err := helper.SendStatusChangeNotification(deviceName, online)
		if err != nil {
			helper.Logger.Error("send device status change notification error", zap.String("deviceName", deviceName))
		}
	})
	// 添加设备
	for _, config := range configMap {
		for _, model := range config.DeviceModels {
			for _, d := range model.Devices {
				dev := shadow.NewDevice(d.Name, model.Name, nil)
				_ = helper.DeviceShadow.AddDevice(dev)
			}
		}
	}
}
