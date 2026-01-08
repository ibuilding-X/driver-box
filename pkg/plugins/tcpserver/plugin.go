package tcpserver

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/pkg/plugins/tcpserver/internal"
)

func RegisterPlugin() {
	plugins0.Manager.Register(internal.ProtocolName, new(internal.Plugin))
}
