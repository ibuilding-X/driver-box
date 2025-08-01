package driverbox

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/bootstrap"
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
)

// ReloadPlugins 重载所有插件
func ReloadPlugins() error {
	return bootstrap.ReloadPlugins()
}

func RegisterPlugin(name string, plugin plugin.Plugin) error {
	return plugins0.Manager.Register(name, plugin)
}
