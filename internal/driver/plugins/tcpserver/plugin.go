package tcpserver

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/contracts"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/driver/common"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"sync"
)

type Plugin struct {
	logger   *zap.Logger
	config   config.Config
	callback contracts.OnReceiveHandler
	adapter  contracts.ProtocolAdapter
	connPool []*connector
	ls       *lua.LState
}

// Initialize 插件初始化
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, handler contracts.OnReceiveHandler, ls *lua.LState) (err error) {
	p.logger = logger
	p.config = c
	p.callback = handler
	p.ls = ls

	// 初始化协议适配器
	p.adapter = &adapter{
		scriptDir: c.Key,
		ls:        ls,
		lock:      &sync.Mutex{},
	}

	// 初始化连接池
	if err = p.initConnPool(); err != nil {
		return
	}

	return nil
}

// ProtocolAdapter 协议适配器
func (p *Plugin) ProtocolAdapter() contracts.ProtocolAdapter {
	return p.adapter
}

// Connector 连接器
func (p *Plugin) Connector(deviceName, pointName string) (connector contracts.Connector, err error) {
	return nil, common.NotSupportGetConnector
}

// Destroy 销毁插件
func (p *Plugin) Destroy() error {
	p.ls.Close()
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
			config: c,
			plugin: p,
		}
		if err = conn.startServer(); err != nil {
			return
		}
		p.connPool = append(p.connPool, conn)
	}
	return
}
