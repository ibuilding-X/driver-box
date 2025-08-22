package plugins

import (
	"github.com/ibuilding-x/driver-box/plugins/bacnet"
	"github.com/ibuilding-x/driver-box/plugins/dlt645"
	"github.com/ibuilding-x/driver-box/plugins/gateway"
	"github.com/ibuilding-x/driver-box/plugins/httpclient"
	"github.com/ibuilding-x/driver-box/plugins/httpserver"
	"github.com/ibuilding-x/driver-box/plugins/mirror"
	"github.com/ibuilding-x/driver-box/plugins/modbus"
	"github.com/ibuilding-x/driver-box/plugins/mqtt"
	"github.com/ibuilding-x/driver-box/plugins/tcpserver"
	"github.com/ibuilding-x/driver-box/plugins/websocket"
)

func RegisterAllPlugins() {
	modbus.RegisterPlugin()
	bacnet.RegisterPlugin()
	httpserver.RegisterPlugin()
	httpclient.RegisterPlugin()
	websocket.RegisterPlugin()
	tcpserver.RegisterPlugin()
	mqtt.RegisterPlugin()
	mirror.RegisterPlugin()
	dlt645.RegisterPlugin()
	gateway.RegisterPlugin()
}
