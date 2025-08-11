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

func RegisterAllPlugins() error {
	if err := modbus.RegisterPlugin(); err != nil {
		return err
	}
	if err := bacnet.RegisterPlugin(); err != nil {
		return err
	}
	if err := httpserver.RegisterPlugin(); err != nil {
		return err
	}
	if err := httpclient.RegisterPlugin(); err != nil {
		return err
	}
	if err := websocket.RegisterPlugin(); err != nil {
		return err
	}
	if err := tcpserver.RegisterPlugin(); err != nil {
		return err
	}
	if err := mqtt.RegisterPlugin(); err != nil {
		return err
	}
	if err := mirror.RegisterPlugin(); err != nil {
		return err
	}
	if err := dlt645.RegisterPlugin(); err != nil {
		return err
	}
	if err := gateway.RegisterPlugin(); err != nil {
		return err
	}
	return nil
}
