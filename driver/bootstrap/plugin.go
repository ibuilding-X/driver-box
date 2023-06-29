package bootstrap

import (
	coreConfig "driver-box/core/config"
	"driver-box/core/contracts"
	"driver-box/core/helper"
	"driver-box/core/helper/crontab"
	"driver-box/core/helper/shadow"
	"driver-box/driver/common"
	"driver-box/driver/device"
	"driver-box/driver/plugins"
	"encoding/json"
	"errors"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"time"
)

// LoadPlugins 加载插件并运行
func LoadPlugins() error {
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
	initDeviceShadow(helper.DriverConfig.DefaultDeviceTTL, configMap)

	// 初始化 crontab
	helper.Crontab = crontab.NewCrontab()

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
			if err = initTimerTasks(ls, configMap[key].Tasks); err != nil {
				return err
			}
		}

		helper.Logger.Info("start success", zap.Any("directoryName", key), zap.Any("plugin", configMap[key].ProtocolName))
	}

	return nil
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

// 初始化设备 autoEvent
func initTimerTasks(L *lua.LState, tasks []coreConfig.TimerTask) (err error) {
	for _, task := range tasks {
		switch task.Type {
		case "read_points": // 读点位
			// action数据格式：[{"devices":["sensor_1"],"points":["onOff"]}]
			b, _ := json.Marshal(task.Action)
			var actions []coreConfig.ReadPointsAction
			_ = json.Unmarshal(b, &actions)
			for _, action := range actions {
				if len(action.DeviceNames) > 0 && len(action.Points) > 0 {
					if err := helper.Crontab.AddFunc(task.Interval+"ms", timerTaskHandler(action.DeviceNames, action.Points)); err != nil {
						return err
					}
				}
			}
		case "script": // 执行脚本函数
			// action数据格式：functionName
			funcName, _ := task.Action.(string)
			if err := helper.Crontab.AddFunc(task.Interval+"ms", timerTaskForScript(L, funcName)); err != nil {
				return err
			}
		}
	}
	helper.Crontab.Start()
	return
}

// timerTaskHandler autoEvent处理函数
func timerTaskHandler(deviceNames []string, pointNames []string) func() {
	return func() {
		helper.Logger.Info("begin handle auto event",
			zap.String("taskType", "read_points"),
			zap.String("deviceNames", fmt.Sprintf("%+v", deviceNames)),
			zap.String("pointNames", fmt.Sprintf("%+v", pointNames)))
		if err := helper.SendMultiRead(deviceNames, pointNames); err != nil {
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
			helper.WriteToMessageBus(data)
		}
		return
	}
}
