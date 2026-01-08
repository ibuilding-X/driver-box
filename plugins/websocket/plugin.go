package websocket

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/websocket/internal"
)

func RegisterPlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
