package mqtt

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

type Plugin struct {
	logger     *zap.Logger           // 日志
	connectors map[string]*connector // mqtt连接池
	ls         *lua.LState           // lua 虚拟机
}

// Initialize 初始化日志、配置、接收回调
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) (err error) {
	p.logger = logger
	p.ls = ls

	// 初始化连接池
	if err = p.initConnPool(c); err != nil {
		return
	}

	return nil
}

func (p *Plugin) initConnPool(c config.Config) error {
	p.connectors = make(map[string]*connector)
	for k, connection := range c.Connections {
		var connectConfig ConnectConfig
		if err := helper.Map2Struct(connection, &connectConfig); err != nil {
			p.logger.Error(fmt.Sprintf("unmarshal mqtt config error: %s", err.Error()))
			continue
		}
		conn := &connector{
			plugin: p,
			config: connectConfig,
		}
		err := conn.connect(connectConfig)
		if err != nil {
			p.logger.Error(fmt.Sprintf("mqtt connect error: %s", err.Error()))
			continue
		}
		p.connectors[k] = conn
	}
	return nil
}

// Connector 连接器
func (p *Plugin) Connector(deviceId, pointName string) (plugin.Connector, error) {
	// 获取连接key
	device, ok := helper.CoreCache.GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connectors[device.ConnectionKey]
	if !ok {
		helper.Logger.Error("not found connection key, key is ", zap.String("key", device.ConnectionKey), zap.Any("connections", p.connectors))
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	return c, nil
}

// Destroy 销毁驱动
func (p *Plugin) Destroy() error {
	if p.ls != nil {
		helper.Close(p.ls)
	}
	connectors := p.connectors
	for _, conn := range connectors {
		conn := conn
		go func() {
			conn.client.Disconnect(0)
		}()
	}
	return nil
}
