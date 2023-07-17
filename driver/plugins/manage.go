// 插件管理器

package plugins

import (
	"driver-box/core/config"
	"driver-box/core/contracts"
	"driver-box/driver/plugins/httpclient"
	"driver-box/driver/plugins/httpserver"
	"driver-box/driver/plugins/modbus"
	"driver-box/driver/plugins/mqtt"
	"driver-box/driver/plugins/tcpserver"
	"fmt"
	"sync"
)

// Manager 插件管理器
var Manager *manager

func init() {
	Manager = &manager{}
	Manager.Register("http_server", new(httpserver.Plugin))
	Manager.Register("modbus", new(modbus.Plugin))
	Manager.Register("tcp_server", new(tcpserver.Plugin))
	Manager.Register("mqtt", new(mqtt.Plugin))
	Manager.Register("http_client", new(httpclient.Plugin))
}

// manager 管理器
type manager struct {
	plugins *sync.Map
}

// 注册自定义插件
func (m *manager) Register(name string, plugin contracts.Plugin) error {
	if _, ok := m.plugins.Load(name); ok {
		panic("plugin:" + name + " is exists")
	} else {
		m.plugins.Store(name, plugin)
	}
	return nil
}

// Get 获取插件实例
func (m *manager) Get(c config.Config) (plugin contracts.Plugin, err error) {
	if raw, ok := m.plugins.Load(c.ProtocolName); ok {
		plugin = raw.(contracts.Plugin)
	} else {
		err = fmt.Errorf("not found drive plugin, plugin name is %s", c.ProtocolName)
	}
	return
}
