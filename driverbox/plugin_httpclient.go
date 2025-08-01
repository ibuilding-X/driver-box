package driverbox

import (
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpclient"
)

func RegisterHttpClientPlugin() error {
	return plugins0.Manager.Register(httpclient.ProtocolName, new(httpclient.Plugin))
}
