package websocket

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/pkg/plugins/websocket/internal"
)

func RegisterPlugin() {
	plugins.Manager.Register(internal.ProtocolName, new(internal.Plugin))
}
