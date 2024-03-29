package bootstrap

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/cmanager"
	"github.com/ibuilding-x/driver-box/driverbox/helper/shadow"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/plugins"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// LoadPlugins 加载插件并运行
func LoadPlugins() error {
	//打印环境配置
	helper.Logger.Info("driver-box environment config", zap.Any("config", helper.EnvConfig))
	// 加载核心配置
	err := cmanager.Load(helper.EnvConfig.ConfigPath)
	if err != nil {
		return errors.New(common.LoadCoreConfigErr.Error() + ":" + err.Error())
	}
	configMap := cmanager.Manager.GetConfigMap()

	if len(configMap) == 0 {
		helper.Logger.Warn("driver-box config is empty", zap.String("path", helper.EnvConfig.ConfigPath))
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

	// 初始化本地影子服务
	initDeviceShadow(configMap)

	// 启动插件
	for key, _ := range configMap {
		helper.Logger.Info(key+" begin start", zap.Any("directoryName", key), zap.Any("plugin", configMap[key].ProtocolName))
		// 获取插件示例
		p, err := plugins.Manager.Get(configMap[key])
		if err != nil {
			helper.Logger.Error(err.Error())
			continue
		}

		ls, err := helper.InitLuaVM(key)
		if err != nil {
			helper.Logger.Error(err.Error())
			continue
		}

		err = p.Initialize(helper.Logger, configMap[key], receiveHandler(), ls)
		if err != nil {
			return err
		}

		// 缓存插件
		helper.CoreCache.AddRunningPlugin(key, p)

		// 初始化定时任务
		if configMap[key].Tasks != nil && len(configMap[key].Tasks) > 0 {
			if err = initTimerTasks(ls, configMap[key].Tasks); err != nil {
				return err
			}
		}

		helper.Logger.Info("start success", zap.Any("directoryName", key), zap.Any("plugin", configMap[key].ProtocolName))
	}

	return nil
}

// 初始化影子服务
func initDeviceShadow(configMap map[string]config.Config) {
	// 设置影子服务设备生命周期
	helper.DeviceShadow = shadow.NewDeviceShadow()
	// 设置回调
	helper.DeviceShadow.SetOnlineChangeCallback(func(deviceSn string, online bool) {
		if online {
			helper.Logger.Info("device online", zap.String("deviceSn", deviceSn))
		} else {
			helper.Logger.Warn("device offline...", zap.String("deviceSn", deviceSn))
		}
		//触发设备在离线事件
		helper.TriggerEvents(event.EventCodeDeviceStatus, deviceSn, online)
	})
	// 添加设备
	for _, c := range configMap {
		for _, model := range c.DeviceModels {
			for _, d := range model.Devices {
				if d.DeviceSn == "" {
					helper.Logger.Error("config error ,device sn is empty", zap.Any("device", d))
					continue
				}
				// 特殊处理：虚拟设备 TTL 值设置
				if c.ProtocolName == "virtual" {
					d.Ttl = "8760h"
				}
				dev := shadow.NewDevice(d, model.Name, nil)
				_ = helper.DeviceShadow.AddDevice(dev)
			}
		}
	}
}

// 初始化设备 autoEvent
func initTimerTasks(L *lua.LState, tasks []config.TimerTask) (err error) {
	//目前看来几乎用不到该功能，先注释掉
	//for _, task := range tasks {
	//	switch task.Type {
	//	case "read_points": // 读点位
	//		// action数据格式：[{"devices":["sensor_1"],"points":["onOff"]}]
	//		b, _ := json.Marshal(task.Action)
	//		var actions []config.ReadPointsAction
	//		_ = json.Unmarshal(b, &actions)
	//		for _, action := range actions {
	//			if len(task.Interval) == 0 {
	//				helper.Logger.Error("timerTask interval is nil")
	//				continue
	//			}
	//			if len(action.Devices) > 0 && len(action.Points) > 0 {
	//				if err := helper.Crontab.AddFunc(task.Interval+"ms", timerTaskHandler(action.Devices, action.Points)); err != nil {
	//					return err
	//				}
	//			}
	//		}
	//	case "script": // 执行脚本函数
	//		// action数据格式：functionName
	//		funcName, _ := task.Action.(string)
	//		if err := helper.Crontab.AddFunc(task.Interval+"ms", timerTaskForScript(L, funcName)); err != nil {
	//			return err
	//		}
	//	}
	//}

	return
}

// timerTaskHandler autoEvent处理函数
func timerTaskHandler(devices []string, pointNames []string) func() {
	return func() {
		helper.Logger.Info("begin handle auto event",
			zap.String("taskType", "read_points"),
			zap.String("devices", fmt.Sprintf("%+v", devices)),
			zap.String("pointNames", fmt.Sprintf("%+v", pointNames)))
		if err := helper.SendMultiRead(devices, pointNames); err != nil {
			helper.Logger.Error("auto event send error", zap.Error(err))
		}
	}
}

// 定时任务 - 执行脚本函数
func timerTaskForScript(L *lua.LState, method string) func() {
	return func() {
		helper.Logger.Info("begin handle auto event", zap.String("taskType", "script"))
		if err := helper.SafeCallLuaFunc(L, method); err != nil {
			helper.Logger.Error("auto event error", zap.Error(err))
		}
	}
}

// receiveHandler 接收消息回调
func receiveHandler() plugin.OnReceiveHandler {
	return func(plugin plugin.Plugin, raw interface{}) (result interface{}, err error) {
		helper.Logger.Debug("raw data", zap.Any("data", raw))
		// 协议适配器
		deviceData, err := plugin.ProtocolAdapter().Decode(raw)
		helper.Logger.Debug("decode data", zap.Any("data", deviceData))
		if err != nil {
			return nil, err
		}
		// 写入消息总线
		for _, data := range deviceData {
			//helper.WriteToMessageBus(data)
			helper.PointCacheFilter(&data)
			if len(data.Values) == 0 {
				continue
			}
			for _, export := range helper.Exports {
				if export.IsReady() {
					export.ExportTo(data)
				}
			}
		}
		return
	}
}
