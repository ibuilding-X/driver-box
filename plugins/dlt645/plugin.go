package dlt645

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/dlt645/internal"
)

func RegisterPlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
