package httpclient

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/pkg/plugins/httpclient/internal"
)

func RegisterPlugin() {
	plugins.Manager.Register(internal.ProtocolName, new(internal.Plugin))
}
