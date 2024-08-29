package driverbox

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/bootstrap"
	"go.uber.org/zap"
	"sync"
)

// reloadLock 用于控制 plugin 重载的互斥锁
var reloadLock sync.Mutex

// ReloadPlugins 重载所有插件
func ReloadPlugins() error {
	reloadLock.Lock()
	defer reloadLock.Unlock()

	helper.Logger.Info("reload all plugins")

	// 1. 停止所有 timerTask 任务
	helper.Crontab.Stop()
	// 2. 停止运行中的 plugin
	pluginKeys := helper.CoreCache.GetAllRunningPluginKey()
	if len(pluginKeys) > 0 {
		for i, _ := range pluginKeys {
			if plugin, ok := helper.CoreCache.GetRunningPluginByKey(pluginKeys[i]); ok {
				err := plugin.Destroy()
				if err != nil {
					helper.Logger.Error("stop plugin error", zap.String("plugin", pluginKeys[i]), zap.Error(err))
				} else {
					helper.Logger.Info("stop plugin success", zap.String("plugin", pluginKeys[i]))
				}
			}
		}
	}
	// 3. 停止影子服务设备状态监听、删除影子服务
	helper.DeviceShadow.StopStatusListener()
	helper.DeviceShadow = nil
	// 4. 清除核心缓存数据
	helper.CoreCache.Reset()
	// 5. 加载 plugins
	return bootstrap.LoadPlugins()
}
