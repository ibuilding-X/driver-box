package driverbox

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/websocket"
)

func RegisterWebsocketPlugin() error {
	return plugins0.Manager.Register(websocket.ProtocolName, new(websocket.Plugin))
}
