package plugin

import (
	"errors"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/ibuilding-x/driver-box/v2/pkg/convutil"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// ProtocolName 插件名称
const ProtocolName = "driverbox"

type gatewayPlugin struct {
	l           *zap.Logger
	c           config.DeviceConfig
	ls          *lua.LState
	connections map[string]*connector
}

func New() plugin.Plugin {
	return &gatewayPlugin{
		connections: make(map[string]*connector),
	}
}

func (g *gatewayPlugin) Initialize(c config.DeviceConfig) {
	g.c = c

	// 初始化连接
	if err := g.initConnection(); err != nil {
		g.l.Error("init connection failed", zap.Error(err))
	}
}

func (g *gatewayPlugin) Connector(deviceId string) (connector plugin.Connector, err error) {
	// 获取连接 key
	device, ok := driverbox.CoreCache().GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := g.connections[device.ConnectionKey]
	if !ok {
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	return c, nil
}

// Destroy 释放所有 ws 连接资源
func (g *gatewayPlugin) Destroy() error {
	if len(g.connections) > 0 {
		for i, _ := range g.connections {
			g.connections[i].destroyed = true
			// 关闭 ws 连接
			if g.connections[i].conn != nil {
				_ = g.connections[i].conn.Close()
			}
		}
	}

	return nil
}

func (g *gatewayPlugin) initConnection() error {
	if len(g.c.Connections) > 0 {
		for connKey, _ := range g.c.Connections {
			conf := &connectorConfig{}
			if err := convutil.Struct(g.c.Connections[connKey], conf); err != nil {
				return err
			}

			// 检查配置项
			if err := conf.checkAndRepair(); err != nil {
				return err
			}

			c := &connector{
				conf: *conf,
			}

			go c.connect()
			g.connections[connKey] = c
		}
	}

	return nil
}
