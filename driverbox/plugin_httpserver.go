package driverbox

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpserver"
)

func RegisterHttpServerPlugin() error {
	return plugins0.Manager.Register(httpserver.ProtocolName, new(httpserver.Plugin))
}
