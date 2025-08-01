package modbus

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/modbus"
)

func RegisterPlugin() error {
	return plugins0.Manager.Register(modbus.ProtocolName, new(modbus.Plugin))
}
