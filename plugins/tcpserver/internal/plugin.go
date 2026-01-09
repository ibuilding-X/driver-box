package internal

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/common"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
)

const ProtocolName = "tcp_server"

type Plugin struct {
	config   config.Config
	connPool []*connector
}

// Initialize 插件初始化
func (p *Plugin) Initialize(c config.Config) {
	p.config = c

	// 初始化连接池
	if err := p.initConnPool(); err != nil {
		helper.Logger.Error("init connector pool failed", zap.Error(err))
	}

}

// Connector 连接器
func (p *Plugin) Connector(deviceSn string) (connector plugin.Connector, err error) {
	return nil, common.NotSupportGetConnector
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
		if err = helper.Map2Struct(p.config.Connections[key], &c); err != nil {
			return
		}
		conn := &connector{
			config:    c,
			plugin:    p,
			scriptDir: p.config.Key,
		}
		if err = conn.startServer(); err != nil {
			return
		}
		p.connPool = append(p.connPool, conn)
	}
	return
}
