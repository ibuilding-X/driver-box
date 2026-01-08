package internal

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox/common"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

const ProtocolName = "tcp_server"

type Plugin struct {
	logger   *zap.Logger
	config   config.Config
	connPool []*connector
	ls       *lua.LState
}

// Initialize 插件初始化
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) {
	p.logger = logger
	p.config = c
	p.ls = ls

	// 初始化连接池
	if err := p.initConnPool(); err != nil {
		logger.Error("init connector pool failed", zap.Error(err))
	}

}

// Connector 连接器
func (p *Plugin) Connector(deviceSn string) (connector plugin.Connector, err error) {
	return nil, common.NotSupportGetConnector
}

// Destroy 销毁插件
func (p *Plugin) Destroy() error {
	if p.ls != nil {
		helper.Close(p.ls)
	}
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
			ls:        p.ls,
		}
		if err = conn.startServer(); err != nil {
			return
		}
		p.connPool = append(p.connPool, conn)
	}
	return
}
