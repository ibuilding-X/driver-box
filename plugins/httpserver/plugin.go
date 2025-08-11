package httpserver

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpserver"
)

func RegisterPlugin() {
	plugins.Manager.Register(httpserver.ProtocolName, new(httpserver.Plugin))
}
