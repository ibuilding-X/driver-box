package mqtt

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/mqtt/internal"
)

func EnablePlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
