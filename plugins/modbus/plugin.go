package modbus

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/modbus/internal"
)

func RegisterPlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
