package httpclient

import (
	"github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpclient"
)

func RegisterPlugin() error {
	return plugins.Manager.Register(httpclient.ProtocolName, new(httpclient.Plugin))
}
