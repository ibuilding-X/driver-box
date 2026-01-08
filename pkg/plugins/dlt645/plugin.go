package dlt645

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
	"github.com/ibuilding-x/driver-box/pkg/plugins/dlt645/internal"
)

func RegisterPlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
