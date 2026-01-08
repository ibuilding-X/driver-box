// 核心工具助手文件

package helper

import (
	"encoding/json"

	"github.com/ibuilding-x/driver-box/internal/core/cache"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/pkg/shadow"

	"sync"
)

var DeviceShadow shadow.DeviceShadow // 本地设备影子
// CoreCache 核心缓存
var CoreCache cache.CoreCache
var PluginCacheMap = &sync.Map{} // 插件通用缓存

var Crontab crontab.Crontab // 全局定时任务实例

var EnvConfig config.EnvConfig

// Map2Struct map 转 struct，用于解析连接器配置
// m：map[string]interface
// v：&struct{}
func Map2Struct(m interface{}, v interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
