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

// plugins 插件管理器，用于统一管理所有注册的插件
var plugins *manager

func init() {
	plugins = &manager{
		plugins: make(map[string]plugin.Plugin, 0),
	}
}

// manager 插件管理器结构体，负责插件的注册、启动、停止等生命周期管理
type manager struct {
	plugins map[string]plugin.Plugin // 存储已注册的插件，key为插件名称，value为插件实例
}

// register 注册自定义插件到管理器中
// 如果插件已存在，则会替换原有插件
// 参数:
//   - name: 插件名称，作为唯一标识符
//   - plugin: 插件实例，需要实现 plugin.Plugin 接口
func (m *manager) register(name string, plugin plugin.Plugin) {
	if _, ok := m.plugins[name]; ok {
		fmt.Printf("plugin %s already exists, replace it", name)
	}
	fmt.Printf("register plugin: %s\n", name)
	m.plugins[name] = plugin
}

// clear 清空所有已注册的插件
// 主要用于插件重新加载时清理旧的插件实例
func (m *manager) clear() {
	m.plugins = make(map[string]plugin.Plugin, 0)
}

// loadPlugins 加载所有已注册的插件并启动它们
// 该函数会执行以下操作：
// 1. 卸载设备驱动库
// 2. 初始化核心缓存
// 3. 初始化设备影子服务
// 4. 启动所有已注册的插件
// 5. 触发设备添加事件通知
// 返回值:
//   - error: 操作过程中出现的错误，如果成功则返回 nil
func loadPlugins() error {
	//清空设备驱动库
	library.Driver().UnloadDeviceDrivers()
	//清空协议驱动库
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

// initDeviceShadow 初始化设备影子服务
// 遍历所有缓存中的设备，将它们添加到影子服务中
// 如果设备已经存在于影子服务中，则跳过该设备
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

// destroyPlugins 销毁所有已启动的插件
// 调用每个插件的 Destroy 方法来释放资源
// 在程序关闭时调用此函数以确保插件正确清理资源
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

// EnablePlugin 注册插件到系统中
// 这是外部插件启用的主要入口点
// 参数:
//   - name: 插件名称，必须唯一，建议使用协议名称
//   - plugin: 实现了 plugin.Plugin 接口的插件实例
//
// 使用示例:
//
//	func EnablePlugin() {
//	    driverbox.EnablePlugin("modbus", new(modbus.Plugin))
//	}
func EnablePlugin(name string, plugin plugin.Plugin) {
	plugins.register(name, plugin)
}

// ReloadPlugin 重启指定名称的插件
// 先调用插件的 Destroy 方法释放资源，然后使用原始配置重新初始化
// 参数:
//   - pluginName: 需要重启的插件名称
//
// 此函数主要用于热重载插件配置或修复插件状态异常
func ReloadPlugin(pluginName string) {
	cfg := cache.GetConfig(pluginName)
	p := plugins.plugins[pluginName]
	p.Destroy()
	p.Initialize(cfg)
}

// ReloadPlugins 重启所有已注册的插件
// 依次对每个插件调用 ReloadPlugin 函数
//
// 此函数主要用于系统级别的插件批量重启，例如在全局配置变更后
func ReloadPlugins() {
	for name, _ := range plugins.plugins {
		ReloadPlugin(name)
	}
}