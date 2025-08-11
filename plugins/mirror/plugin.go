package mirror

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/mirror"
)

func RegisterPlugin() {
	plugins0.Manager.Register(mirror.ProtocolName, mirror.NewPlugin())
}
