package tcpserver

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/tcpserver/internal"
)

func RegisterPlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
