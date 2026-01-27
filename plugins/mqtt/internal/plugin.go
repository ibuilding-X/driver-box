package internal

import (
	"errors"
	"fmt"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"go.uber.org/zap"
)

const ProtocolName = "mqtt"

type Plugin struct {
	connectors map[string]*connector // mqtt连接池
}

// Initialize 初始化日志、配置、接收回调
func (p *Plugin) Initialize(c config.DeviceConfig) {
	// 初始化连接池
	go func() {
		if err := p.initConnPool(c); err != nil {
			driverbox.Log().Error("init mqtt connector error", zap.Error(err))
		}
	}()
}

func (p *Plugin) initConnPool(c config.DeviceConfig) error {
	p.connectors = make(map[string]*connector)
	for k, connection := range c.Connections {
		var connectConfig ConnectConfig
		if err := convutil.Struct(connection, &connectConfig); err != nil {
			driverbox.Log().Error(fmt.Sprintf("unmarshal mqtt config error: %s", err.Error()))
			continue
		}
		// 删除不支持自动发现，且未关联设备得连接
		if !connectConfig.Discover && !config.HasDevice(k, c) {
			err := driverbox.CoreCache().DeleteConnection(k)
			if err != nil {
				driverbox.Log().Error("delete connection error", zap.Any("connection", connectConfig), zap.Error(err))
			} else {
				driverbox.Log().Warn("delete connection success", zap.Any("connection", connectConfig))
			}
			continue
		}

		if !connectConfig.Enable {
			driverbox.Log().Warn("mqtt connection is disable", zap.String("connectionKey", k))
			continue
		}
		connectConfig.ConnectionKey = k
		conn := &connector{
			config: connectConfig,
		}
		err := conn.connect(connectConfig)
		if err != nil {
			driverbox.Log().Error(fmt.Sprintf("mqtt connect error: %s", err.Error()))
			continue
		}
		p.connectors[k] = conn
	}
	return nil
}

// Connector 连接器
func (p *Plugin) Connector(deviceId string) (plugin.Connector, error) {
	// 获取连接key
	device, ok := driverbox.CoreCache().GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connectors[device.ConnectionKey]
	if !ok {
		driverbox.Log().Error("not found connection key, key is ", zap.String("key", device.ConnectionKey), zap.Any("connections", p.connectors))
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
