package driverbox

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/dlt645"
)

func RegisterDlt645Plugin() error {
	return plugins0.Manager.Register(dlt645.ProtocolName, new(dlt645.Plugin))
}
