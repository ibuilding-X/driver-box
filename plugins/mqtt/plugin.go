package mqtt

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/mqtt"
)

func RegisterPlugin() error {
	return plugins0.Manager.Register(mqtt.ProtocolName, new(mqtt.Plugin))
}
