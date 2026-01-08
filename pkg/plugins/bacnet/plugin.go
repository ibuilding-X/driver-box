package bacnet

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
	"github.com/ibuilding-x/driver-box/pkg/plugins/bacnet/internal"
)

func RegisterPlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
