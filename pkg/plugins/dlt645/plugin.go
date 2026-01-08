package dlt645

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/pkg/plugins/dlt645/internal"
)

func RegisterPlugin() {
	plugins.Manager.Register(internal.ProtocolName, new(internal.Plugin))
}
