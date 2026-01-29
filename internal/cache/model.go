package cache

import (
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
)

type cachePlugin struct {
	plugin plugin.Plugin
	// 配置文件路径
	FilePath string `json:"-" validate:"-"`

	fileModifyTime  time.Time
	cacheModifyTime time.Time
}

type cacheConnection struct {
	pluginName string
	connection any
}
type cacheModel struct {
	config.Model
	pluginName string
	points     map[string]*config.Point
}

type cacheDevice struct {
	config.Device
}
