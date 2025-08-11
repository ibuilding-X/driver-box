package bacnet

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet"
)

func RegisterPlugin() {
	plugins.Manager.Register(bacnet.ProtocolName, new(bacnet.Plugin))
}
