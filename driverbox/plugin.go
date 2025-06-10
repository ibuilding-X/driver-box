package driverbox

import (
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/bootstrap"
	plugins0 "github.com/ibuilding-x/driver-box/internal/plugins"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet"
	"github.com/ibuilding-x/driver-box/internal/plugins/dlt645"
	"github.com/ibuilding-x/driver-box/internal/plugins/gwplugin"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpclient"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpserver"
	"github.com/ibuilding-x/driver-box/internal/plugins/mirror"
	"github.com/ibuilding-x/driver-box/internal/plugins/modbus"
	"github.com/ibuilding-x/driver-box/internal/plugins/mqtt"
	"github.com/ibuilding-x/driver-box/internal/plugins/tcpserver"
	"github.com/ibuilding-x/driver-box/internal/plugins/websocket"
	"sync"
)

var Plugins plugins

type plugins struct {
}

// reloadLock 用于控制 plugin 重载的互斥锁
var reloadLock sync.Mutex

// ReloadPlugins 重载所有插件
func (p *plugins) ReloadPlugins() error {
	return bootstrap.ReloadPlugins()
}

func (p *plugins) RegisterPlugin(name string, plugin plugin.Plugin) error {
	return plugins0.Manager.Register(name, plugin)
}

func (p *plugins) RegisterAllPlugins() error {
	if err := p.RegisterModbusPlugin(); err != nil {
		return err
	}
	if err := p.RegisterBacnetPlugin(); err != nil {
		return err
	}
	if err := p.RegisterHttpServerPlugin(); err != nil {
		return err
	}
	if err := p.RegisterHttpClientPlugin(); err != nil {
		return err
	}
	if err := p.RegisterWebsocketPlugin(); err != nil {
		return err
	}
	if err := p.RegisterTcpServerPlugin(); err != nil {
		return err
	}
	if err := p.RegisterMqttPlugin(); err != nil {
		return err
	}
	if err := p.RegisterMirrorPlugin(); err != nil {
		return err
	}
	if err := p.RegisterDlt645Plugin(); err != nil {
		return err
	}
	if err := p.RegisterGatewayPlugin(); err != nil {
		return err
	}
	return nil
}

func (p *plugins) RegisterModbusPlugin() error {
	return plugins0.Manager.Register(modbus.ProtocolName, new(modbus.Plugin))
}

func (p *plugins) RegisterBacnetPlugin() error {
	return plugins0.Manager.Register(bacnet.ProtocolName, new(bacnet.Plugin))
}

func (p *plugins) RegisterHttpServerPlugin() error {
	return plugins0.Manager.Register(httpserver.ProtocolName, new(httpserver.Plugin))
}

func (p *plugins) RegisterHttpClientPlugin() error {
	return plugins0.Manager.Register(httpclient.ProtocolName, new(httpclient.Plugin))
}

func (p *plugins) RegisterWebsocketPlugin() error {
	return plugins0.Manager.Register(websocket.ProtocolName, new(websocket.Plugin))
}

func (p *plugins) RegisterTcpServerPlugin() error {
	return plugins0.Manager.Register(tcpserver.ProtocolName, new(tcpserver.Plugin))
}

func (p *plugins) RegisterMqttPlugin() error {
	return plugins0.Manager.Register(mqtt.ProtocolName, new(mqtt.Plugin))
}

func (p *plugins) RegisterMirrorPlugin() error {
	return plugins0.Manager.Register(mirror.ProtocolName, mirror.NewPlugin())
}

func (p *plugins) RegisterDlt645Plugin() error {
	return plugins0.Manager.Register(dlt645.ProtocolName, new(dlt645.Plugin))
}

func (p *plugins) RegisterGatewayPlugin() error {
	return plugins0.Manager.Register(gwplugin.ProtocolName, gwplugin.New())
}
