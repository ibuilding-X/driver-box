package tcpserver

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/tcpserver"
)

func RegisterPlugin() {
	plugins0.Manager.Register(tcpserver.ProtocolName, new(tcpserver.Plugin))
}
