package gateway

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox"
	"github.com/ibuilding-x/driver-box/pkg/plugins/gateway/internal"
)

func RegisterPlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, internal.New())
}
