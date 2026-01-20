package cache

import (
	"time"

	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
)

type configCache struct {
	Models map[string]config.DeviceModel
	// 连接配置
	Connections map[string]interface{}
	plugin      plugin.Plugin
	// 配置文件路径
	FilePath string `json:"-" validate:"-"`

	fileModifyTime  time.Time
	cacheModifyTime time.Time
}
