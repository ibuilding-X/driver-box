package httpclient

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/plugins/httpclient/internal"
)

func EnablePlugin() {
	driverbox.RegisterPlugin(internal.ProtocolName, new(internal.Plugin))
}
