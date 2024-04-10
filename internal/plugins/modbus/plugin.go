package modbus

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// Plugin 驱动插件
type Plugin struct {
	logger   *zap.Logger            // 日志记录器
	config   config.Config          // 核心配置
	adapter  plugin.ProtocolAdapter // 协议适配器
	connPool map[string]*connector  // 连接器
	ls       *lua.LState            // lua 虚拟机
}

// Initialize 插件初始化
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) (err error) {
	p.logger = logger
	p.config = c
	p.ls = ls

	// 初始化协议适配器
	p.adapter = &adapter{
		ls:           ls,
		scriptEnable: helper.ScriptExists(c.Key),
	}
	//初始化连接池
	return p.initNetworks()
}

// 初始化Modbus连接池
func (p *Plugin) initNetworks() (err error) {
	p.connPool = make(map[string]*connector)

	for key, connConfig := range p.config.Connections {
		cc := new(connectorConfig)
		if err = helper.Map2Struct(connConfig, cc); err != nil {
			return fmt.Errorf("convert connector config error: %v", err)
		}
		c2, err := newConnector(p, cc)
		if err != nil {
			return fmt.Errorf("init connector error: %v", err)
		}
		p.connPool[key] = c2
	}
	return nil
}

// ProtocolAdapter 适配器
func (p *Plugin) ProtocolAdapter() plugin.ProtocolAdapter {
	return p.adapter
}

// Connector 连接器
func (p *Plugin) Connector(deviceSn, pointName string) (conn plugin.Connector, err error) {
	// 获取连接key
	device, ok := helper.CoreCache.GetDeviceByDeviceAndPoint(deviceSn, pointName)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connPool[device.ConnectionKey]
	if !ok {
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	c.mutex.Lock()
	err = c.client.Open()
	if err != nil {
		c.mutex.Unlock()
		return nil, err
	}
	return c, nil
}

// Destroy 销毁驱动插件
func (p *Plugin) Destroy() error {
	for _, conn := range p.connPool {
		conn.polling = false
	}
	if p.ls != nil {
		p.ls.Close()
	}
	return nil
}
