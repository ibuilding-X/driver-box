// 插件管理器

package plugins

import (
	"driver-box/core/config"
	"driver-box/core/contracts"
	"driver-box/driver/plugins/bacnet"
	"driver-box/driver/plugins/httpclient"
	"driver-box/driver/plugins/httpserver"
	"driver-box/driver/plugins/modbus"
	"driver-box/driver/plugins/mqtt"
	"driver-box/driver/plugins/tcpserver"
	"fmt"
)

// Manager 插件管理器
var Manager *manager

func init() {
	Manager = &manager{}
}

// manager 管理器
type manager struct {
}

// Get 获取插件实例
func (m *manager) Get(c config.Config) (plugin contracts.Plugin, err error) {
	switch c.ProtocolName {
	case "http_server":
		plugin = new(httpserver.Plugin)
	case "modbus":
		plugin = new(modbus.Plugin)
	case "tcp_server":
		plugin = new(tcpserver.Plugin)
	case "mqtt":
		plugin = new(mqtt.Plugin)
	case "http_client":
		plugin = new(httpclient.Plugin)
	default:
		err = fmt.Errorf("not found drive plugin, plugin name is %s", c.ProtocolName)
	}
	return
}
