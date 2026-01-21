package internal

import (
	"context"
	"errors"
	"sync"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"go.uber.org/zap"
)

const ProtocolName = "websocket"

type Plugin struct {
	config config.DeviceConfig // 核心配置

	connPool map[string]*connector // 连接器
}

func (p *Plugin) Initialize(c config.DeviceConfig) {
	p.config = c
	p.connPool = make(map[string]*connector)

	// 初始化连接池
	if err := p.initConnPool(); err != nil {
		driverbox.Log().Error("initialize websocket plugin failed", zap.Error(err))
	}

}

// Connector 此协议不支持获取连接器
func (p *Plugin) Connector(deviceId string) (connector plugin.Connector, err error) {
	// 获取连接key
	device, ok := driverbox.CoreCache().GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connPool[device.ConnectionKey]
	if !ok {
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	return c, nil
}

func (p *Plugin) Destroy() error {
	if len(p.connPool) > 0 {
		for _, c := range p.connPool {
			if c.server != nil {
				_ = c.server.Shutdown(context.Background())
			}
		}
	}
	return nil
}

// initConnPool 初始化连接池
func (p *Plugin) initConnPool() (err error) {
	for key, _ := range p.config.Connections {
		var c connectorConfig
		if err = convutil.Struct(p.config.Connections[key], &c); err != nil {
			return
		}
		c.ConnectionKey = key
		conn := &connector{
			config:            c,
			deviceMappingConn: &sync.Map{},
			connMappingDevice: &sync.Map{},
		}
		conn.startServer()
		p.connPool[key] = conn
	}
	return
}
