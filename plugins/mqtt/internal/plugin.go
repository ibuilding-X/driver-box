package internal

import (
	"errors"
	"fmt"

	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/config"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/convutil"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
)

const ProtocolName = "mqtt"

type Plugin struct {
	connectors map[string]*connector // mqtt连接池
}

// Initialize 初始化日志、配置、接收回调
func (p *Plugin) Initialize(c config.Config) {

	// 初始化连接池
	if err := p.initConnPool(c); err != nil {
		helper.Logger.Error("init mqtt connector error", zap.Error(err))
	}

}

func (p *Plugin) initConnPool(c config.Config) error {
	p.connectors = make(map[string]*connector)
	for k, connection := range c.Connections {
		var connectConfig ConnectConfig
		if err := convutil.Struct(connection, &connectConfig); err != nil {
			helper.Logger.Error(fmt.Sprintf("unmarshal mqtt config error: %s", err.Error()))
			continue
		}
		if !connectConfig.Enable {
			helper.Logger.Warn("mqtt connection is disable", zap.String("connectionKey", k))
			continue
		}
		connectConfig.ConnectionKey = k
		conn := &connector{
			plugin: p,
			config: connectConfig,
		}
		err := conn.connect(connectConfig)
		if err != nil {
			helper.Logger.Error(fmt.Sprintf("mqtt connect error: %s", err.Error()))
			continue
		}
		p.connectors[k] = conn
	}
	return nil
}

// Connector 连接器
func (p *Plugin) Connector(deviceId string) (plugin.Connector, error) {
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
	connectors := p.connectors
	for _, conn := range connectors {
		conn := conn
		go func() {
			conn.client.Disconnect(0)
		}()
	}
	return nil
}
