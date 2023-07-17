package modbus

import (
	"errors"
	"github.com/ibuilding-x/driver-box/core/config"
	"github.com/ibuilding-x/driver-box/core/contracts"
	"github.com/ibuilding-x/driver-box/core/helper"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// Plugin 驱动插件
type Plugin struct {
	logger   *zap.Logger                // 日志记录器
	config   config.Config              // 核心配置
	callback contracts.OnReceiveHandler // 回调函数
	adapter  contracts.ProtocolAdapter  // 协议适配器
	connPool map[string]*connector      // 连接器
	ls       *lua.LState                // lua 虚拟机
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
	}

	// 初始化连接池
	if err = p.initConnPool(); err != nil {
		return
	}

	return nil
}

// ProtocolAdapter 适配器
func (p *Plugin) ProtocolAdapter() contracts.ProtocolAdapter {
	return p.adapter
}

// Connector 连接器
func (p *Plugin) Connector(deviceName, pointName string) (conn contracts.Connector, err error) {
	// 获取连接key
	device, ok := helper.CoreCache.GetDeviceByDeviceAndPoint(deviceName, pointName)
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
	if p.ls != nil {
		p.ls.Close()
	}
	return nil
}

// initConnPool 初始化连接池
func (p *Plugin) initConnPool() (err error) {
	p.connPool = make(map[string]*connector)
	for key, _ := range p.config.Connections {
		var c connectorConfig
		if err = helper.Map2Struct(p.config.Connections[key], &c); err != nil {
			return
		}
		conn := &connector{
			plugin: p,
			config: c,
		}
		conn.connect()
		p.connPool[key] = conn
	}
	return
}
