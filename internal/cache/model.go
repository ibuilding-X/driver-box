package cache

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
)

type configs struct {
	Models map[string]config.DeviceModel
	// 连接配置
	Connections map[string]interface{}
	plugin      plugin.Plugin
	// 配置文件路径
	FilePath string `json:"-" validate:"-"`
}

type deviceCache struct {
	config.Device
	protocolName string
}
