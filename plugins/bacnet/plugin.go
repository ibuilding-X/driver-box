package bacnet

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal"
)

func EnablePlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
