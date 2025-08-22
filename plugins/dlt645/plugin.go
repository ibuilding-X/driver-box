package dlt645

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/dlt645"
)

func RegisterPlugin() {
	plugins.Manager.Register(dlt645.ProtocolName, new(dlt645.Plugin))
}
