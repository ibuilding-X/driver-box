// 插件管理器

package plugins

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet"
	"github.com/ibuilding-x/driver-box/internal/plugins/dlt645"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpclient"
	"github.com/ibuilding-x/driver-box/internal/plugins/httpserver"
	"github.com/ibuilding-x/driver-box/internal/plugins/mirror"
	"github.com/ibuilding-x/driver-box/internal/plugins/modbus"
	"github.com/ibuilding-x/driver-box/internal/plugins/mqtt"
	"github.com/ibuilding-x/driver-box/internal/plugins/tcpserver"
	"github.com/ibuilding-x/driver-box/internal/plugins/websocket"
	"sync"
)

// Manager 插件管理器
var Manager *manager

func init() {
	Manager = &manager{
		plugins: &sync.Map{},
	}
	Manager.Register("http_server", new(httpserver.Plugin))
	Manager.Register("modbus", new(modbus.Plugin))
	Manager.Register("tcp_server", new(tcpserver.Plugin))
	Manager.Register("mqtt", new(mqtt.Plugin))
	Manager.Register(httpclient.ProtocolName, new(httpclient.Plugin))
	//Manager.Register("virtual", new(virtual.Plugin))
	Manager.Register("bacnet", new(bacnet.Plugin))
	Manager.Register(websocket.ProtocolName, new(websocket.Plugin))
	Manager.Register(mirror.ProtocolName, mirror.NewPlugin())
	Manager.Register(dlt645.ProtocolName, new(dlt645.Plugin))
}

// manager 管理器
type manager struct {
	plugins *sync.Map
}

// 注册自定义插件
func (m *manager) Register(name string, plugin plugin.Plugin) error {
	if _, ok := m.plugins.Load(name); ok {
		panic("plugin:" + name + " is exists")
	} else {
		m.plugins.Store(name, plugin)
	}
	return nil
}

// Get 获取插件实例
func (m *manager) Get(c config.Config) (p plugin.Plugin, err error) {
	if raw, ok := m.plugins.Load(c.ProtocolName); ok {
		p = raw.(plugin.Plugin)
	} else {
		err = fmt.Errorf("not found drive plugin, plugin name is %s", c.ProtocolName)
	}
	return
}
