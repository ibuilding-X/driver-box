package cache

import (
	"time"

	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
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
	points     map[string]cachePoint
}

type cachePoint struct {
	config.Point
	order int
}

type cacheDevice struct {
	config.Device
}
