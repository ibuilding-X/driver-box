package plugins

import (
	"github.com/ibuilding-x/driver-box/pkg/plugins/bacnet"
	"github.com/ibuilding-x/driver-box/pkg/plugins/dlt645"
	"github.com/ibuilding-x/driver-box/pkg/plugins/gateway"
	"github.com/ibuilding-x/driver-box/pkg/plugins/httpclient"
	"github.com/ibuilding-x/driver-box/pkg/plugins/httpserver"
	"github.com/ibuilding-x/driver-box/pkg/plugins/mirror"
	"github.com/ibuilding-x/driver-box/pkg/plugins/modbus"
	"github.com/ibuilding-x/driver-box/pkg/plugins/mqtt"
	"github.com/ibuilding-x/driver-box/pkg/plugins/tcpserver"
	"github.com/ibuilding-x/driver-box/pkg/plugins/websocket"
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
