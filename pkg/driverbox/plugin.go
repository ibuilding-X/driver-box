package driverbox

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox/internal/bootstrap"
	plugins0 "github.com/ibuilding-x/driver-box/pkg/driverbox/internal/plugins"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
)

// ReloadPlugins 重载所有插件
func ReloadPlugins() error {
	return bootstrap.ReloadPlugins()
}

func RegisterPlugin(name string, plugin plugin.Plugin) {
	plugins0.Manager.Register(name, plugin)
}
