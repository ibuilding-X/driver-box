// 核心工具助手文件

package helper

import (
	"github.com/ibuilding-x/driver-box/driverbox/shadow"
	"github.com/ibuilding-x/driver-box/internal/core/cache"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/crontab"
)

var DeviceShadow shadow.DeviceShadow // 本地设备影子
// CoreCache 核心缓存
var CoreCache cache.CoreCache

var Crontab crontab.Crontab // 全局定时任务实例

var EnvConfig config.EnvConfig
