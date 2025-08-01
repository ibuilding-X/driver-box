package driverbox

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/modbus"
)

func RegisterModbusPlugin() error {
	return plugins0.Manager.Register(modbus.ProtocolName, new(modbus.Plugin))
}
