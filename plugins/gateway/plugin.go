package gateway

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/gwplugin"
)

func RegisterPlugin() {
	plugins.Manager.Register(gwplugin.ProtocolName, gwplugin.New())
}
