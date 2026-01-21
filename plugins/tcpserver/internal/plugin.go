package internal

import (
	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"go.uber.org/zap"
)

const ProtocolName = "tcp_server"

type Plugin struct {
	config   config.DeviceConfig
	connPool []*connector
}

// Initialize 插件初始化
func (p *Plugin) Initialize(c config.DeviceConfig) {
	p.config = c

	// 初始化连接池
	if err := p.initConnPool(); err != nil {
		driverbox.Log().Error("init connector pool failed", zap.Error(err))
	}

}

// Connector 连接器
func (p *Plugin) Connector(deviceSn string) (connector plugin.Connector, err error) {
	return nil, plugin.NotSupportGetConnector
}

// Destroy 销毁插件
func (p *Plugin) Destroy() error {
	return nil
}

// initConnPool 初始化连接池
func (p *Plugin) initConnPool() (err error) {
	p.connPool = make([]*connector, 0)
	for key, _ := range p.config.Connections {
		var c connectorConfig
		if err = convutil.Struct(p.config.Connections[key], &c); err != nil {
			return
		}
		conn := &connector{
			config:    c,
			plugin:    p,
			scriptDir: c.ConnectionKey,
		}
		if err = conn.startServer(); err != nil {
			return
		}
		p.connPool = append(p.connPool, conn)
	}
	return
}
