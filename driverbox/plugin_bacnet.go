package driverbox

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet"
)

func RegisterBacnetPlugin() error {
	return plugins0.Manager.Register(bacnet.ProtocolName, new(bacnet.Plugin))
}
