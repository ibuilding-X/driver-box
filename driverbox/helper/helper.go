// 核心工具助手文件

package helper

import (
	"github.com/ibuilding-x/driver-box/driverbox/internal/core/cache"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/config"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/shadow"

	"sync"
)

var DeviceShadow shadow.DeviceShadow // 本地设备影子
// CoreCache 核心缓存
var CoreCache cache.CoreCache
var PluginCacheMap = &sync.Map{} // 插件通用缓存

var Crontab crontab.Crontab // 全局定时任务实例

var EnvConfig config.EnvConfig
