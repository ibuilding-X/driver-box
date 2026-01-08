package internal

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox/common"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

const ProtocolName = "bacnet"

type Plugin struct {
	logger   *zap.Logger
	config   config.Config
	connPool map[string]plugin.Connector
	ls       *lua.LState
}

// Initialize 插件初始化
// logger *zap.Logger、ls *lua.LState 参数未来可能会废弃
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) {
	p.logger = logger
	p.config = c
	p.ls = ls

	// 初始化连接
	if err := p.initNetworks(); err != nil {
		logger.Error("initialize bacnet plugin error", zap.Error(err))
	}

}

// Connector 连接器
func (p *Plugin) Connector(deviceName string) (connector plugin.Connector, err error) {
	if device, ok := helper.CoreCache.GetDevice(deviceName); ok {
		if conn, ok := p.connPool[device.ConnectionKey]; ok {
			return conn, nil
		}
		return nil, common.ConnectorNotFound
	}
	return nil, common.DeviceNotFoundError
}

// Destroy 销毁插件
func (p *Plugin) Destroy() error {
	for _, conn := range p.connPool {
		c := conn.(*connector)
		c.Close()
	}
	if p.ls != nil {
		helper.Close(p.ls)
	}
	return nil
}

// initNetworks 初始化连接池
func (p *Plugin) initNetworks() (err error) {
	p.connPool = make(map[string]plugin.Connector)
	for connName, conn := range p.config.Connections {
		if n, err := initConnector(connName, conn.(map[string]interface{}), p); err == nil {
			p.connPool[connName] = n
		} else {
			return err
		}
	}
	return nil
}
