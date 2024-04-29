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
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"go.uber.org/zap"
)

// LoadPlugins 加载插件并运行
func LoadPlugins() error {
	//打印环境配置
	helper.Logger.Info("driver-box environment config", zap.Any("config", helper.EnvConfig))
	// 加载核心配置
	cmanager.SetConfigPath(helper.EnvConfig.ConfigPath)
	err := cmanager.LoadConfig()
	if err != nil {
		return errors.New(common.LoadCoreConfigErr.Error() + ":" + err.Error())
	}
	configMap := cmanager.GetConfigs()

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

		err = p.Initialize(helper.Logger, configMap[key], ls)
		if err != nil {
			return err
		}

		// 缓存插件
		helper.CoreCache.AddRunningPlugin(key, p)

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
				if d.ID == "" {
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
