package gateway

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/pkg/plugins/gateway/internal"
)

func RegisterPlugin() {
	plugins.Manager.Register(internal.ProtocolName, internal.New())
}
