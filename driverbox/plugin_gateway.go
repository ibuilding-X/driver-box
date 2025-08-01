package driverbox

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/gwplugin"
)

func RegisterGatewayPlugin() error {
	return plugins0.Manager.Register(gwplugin.ProtocolName, gwplugin.New())
}
