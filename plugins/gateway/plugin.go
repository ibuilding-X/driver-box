package gateway

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/gateway/internal"
)

func EnablePlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, internal.New())
}
