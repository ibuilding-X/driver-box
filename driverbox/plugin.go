package driverbox

import (
	"fmt"

	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/cache"
	"github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"github.com/ibuilding-x/driver-box/pkg/library"
	"go.uber.org/zap"
)

// manager 插件管理器
var plugins *manager

func init() {
	plugins = &manager{
		plugins: make(map[string]plugin.Plugin, 0),
	}
}

// manager 管理器
type manager struct {
	plugins map[string]plugin.Plugin
}

// 注册自定义插件
func (m *manager) Register(name string, plugin plugin.Plugin) {
	if _, ok := m.plugins[name]; ok {
		fmt.Printf("plugin %s already exists, replace it", name)
	}
	fmt.Printf("register plugin: %s\n", name)
	m.plugins[name] = plugin
}

func (m *manager) Clear() {
	m.plugins = make(map[string]plugin.Plugin, 0)
}

// loadPlugins 加载插件并运行
func loadPlugins() error {
	//清空设备驱动库
	library.Driver().UnloadDeviceDrivers()
	//清空设备驱动库
	library.Protocol().UnloadDeviceDrivers()

	// 缓存核心配置
	_, err := cache.InitCoreCache(plugins.plugins)
	if err != nil {
		return err
	}
	// 初始化本地影子服务
	initDeviceShadow()

	// 启动插件
	for key, p := range plugins.plugins {
		Log().Info(key+" begin start", zap.Any("directoryName", key), zap.Any("plugin", key))
		p.Initialize(cache.GetConfig(key))

		Log().Info("start success", zap.Any("directoryName", key), zap.Any("plugin", key))
	}

	//完成初始化后触发设备添加事件通知
	for _, device := range cache.Get().Devices() {
		export.TriggerEvents(event.EventCodeAddDevice, device.ID, nil)
	}

	return nil
}

// 初始化影子服务
func initDeviceShadow() {
	// 添加设备
	for _, dev := range CoreCache().Devices() {
		// 设备存在校验
		if Shadow().HasDevice(dev.ID) {
			Log().Warn("device already exist", zap.String("deviceId", dev.ID))
			continue
		}
		// 添加设备
		Shadow().AddDevice(dev.ID, dev.ModelName)
	}
}

func destroyPlugins() {
	for key, p := range plugins.plugins {
		err := p.Destroy()
		if err != nil {
			Log().Error("stop plugin error", zap.String("plugin", key), zap.Error(err))
		} else {
			Log().Info("stop plugin success", zap.String("plugin", key))
		}
	}
}

// 注册插件
func EnablePlugin(name string, plugin plugin.Plugin) {
	plugins.Register(name, plugin)
}

// 插件重启
func ReloadPlugin(pluginName string) {
	cfg := cache.GetConfig(pluginName)
	p := plugins.plugins[pluginName]
	p.Destroy()
	p.Initialize(cfg)
}
