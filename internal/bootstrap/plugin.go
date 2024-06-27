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
	"github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/library"
	"github.com/ibuilding-x/driver-box/internal/lua"
	"github.com/ibuilding-x/driver-box/internal/plugins"
	glua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"path/filepath"
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

	//初始化设备层驱动
	initDeviceDriver(configMap)

	// 启动插件
	for key, _ := range configMap {
		helper.Logger.Info(key+" begin start", zap.Any("directoryName", key), zap.Any("plugin", configMap[key].ProtocolName))
		// 获取插件示例
		p, err := plugins.Manager.Get(configMap[key])
		if err != nil {
			helper.Logger.Error(err.Error())
			continue
		}

		var ls *glua.LState
		path := filepath.Join(helper.EnvConfig.ConfigPath, key, common.LuaScriptName)
		if common.FileExists(path) {
			ls, err = lua.InitLuaVM(path)
			if err != nil {
				helper.Logger.Error(err.Error())
				continue
			}
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

// 初始化设备层驱动
func initDeviceDriver(configMap map[string]config.Config) {
	//清空设备驱动库
	library.Driver().UnloadDeviceDrivers()
	//重新添加
	drivers := make(map[string]string)
	for _, c := range configMap {
		for _, model := range c.DeviceModels {
			for _, d := range model.Devices {
				if len(d.DriverKey) > 0 {
					drivers[d.DriverKey] = d.DriverKey
				}
			}
		}
	}
	for key, _ := range drivers {
		err := library.Driver().LoadLibrary(key)
		if err != nil {
			helper.Logger.Error("load device driver error", zap.String("driverKey", key), zap.Error(err))
		}
	}
}

// 初始化影子服务
func initDeviceShadow(configMap map[string]config.Config) {
	// 设置影子服务设备生命周期
	if helper.DeviceShadow == nil {
		helper.DeviceShadow = shadow.NewDeviceShadow()
		// 设置回调
		helper.DeviceShadow.SetOnlineChangeCallback(func(deviceId string, online bool) {
			if online {
				helper.Logger.Info("device online", zap.String("deviceId", deviceId))
			} else {
				helper.Logger.Warn("device offline...", zap.String("deviceId", deviceId))
			}
			//触发设备在离线事件
			export.TriggerEvents(event.EventCodeDeviceStatus, deviceId, online)
		})
	}
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
				// 设备存在校验
				if helper.DeviceShadow.HasDevice(d.ID) {
					helper.Logger.Warn("device already exist", zap.String("deviceId", d.ID))
					continue
				}
				// 添加设备
				helper.DeviceShadow.AddDevice(d.ID, model.Name)
			}
		}
	}
}
