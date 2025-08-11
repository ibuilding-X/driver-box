package websocket

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/websocket"
)

func RegisterPlugin() error {
	return plugins.Manager.Register(websocket.ProtocolName, new(websocket.Plugin))
}
