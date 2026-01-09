package internal

import (
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/common"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
)

const ProtocolName = "http_server"

type Plugin struct {
	config config.Config // 核心配置

	connPool []*connector // 连接器
}

func (p *Plugin) Initialize(c config.Config) {
	p.config = c

	// 初始化连接池
	if err := p.initConnPool(); err != nil {
		helper.Logger.Error("init connector pool failed", zap.Error(err))
	}

}

// Connector 此协议不支持获取连接器
func (p *Plugin) Connector(deviceSn string) (connector plugin.Connector, err error) {
	return nil, common.NotSupportGetConnector
}

func (p *Plugin) Destroy() error {
	if len(p.connPool) > 0 {
		for i, _ := range p.connPool {
			if err := p.connPool[i].Release(); err != nil {
				return err
			}
		}
	}
	return nil
}

// initConnPool 初始化连接池
func (p *Plugin) initConnPool() (err error) {
	for key, _ := range p.config.Connections {
		var c connectorConfig
		if err = helper.Map2Struct(p.config.Connections[key], &c); err != nil {
			return
		}
		conn := &connector{
			plugin:      p,
			protocolKey: c.ProtocolKey,
		}
		conn.startServer(c)
		p.connPool = append(p.connPool, conn)
	}
	return
}
