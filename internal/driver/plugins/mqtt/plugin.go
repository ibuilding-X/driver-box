package mqtt

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/contracts"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/driver/common"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"sync"
)

type Plugin struct {
	logger     *zap.Logger                // 日志
	config     config.Config              // 配置
	callback   contracts.OnReceiveHandler // 回调
	adapter    *adapter                   // 适配
	connectors map[string]*connector      // mqtt连接池
	ls         *lua.LState                // lua 虚拟机
}

// Initialize 初始化日志、配置、接收回调
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

func (p *Plugin) initConnPool() error {
	p.connectors = make(map[string]*connector)
	connections := p.config.Connections
	for k, connection := range connections {
		var connectConfig ConnectConfig
		if err := helper.Map2Struct(connection, &connectConfig); err != nil {
			p.logger.Error(fmt.Sprintf("unmarshal mqtt config error: %s", err.Error()))
			continue
		}
		conn := &connector{
			plugin: p,
			topics: connectConfig.Topics,
			name:   k,
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

// ProtocolAdapter 协议适配器
func (p *Plugin) ProtocolAdapter() contracts.ProtocolAdapter {
	return p.adapter
}

// Connector 连接器
func (p *Plugin) Connector(deviceName, pointName string) (contracts.Connector, error) {
	deviceModels := p.config.DeviceModels
	for _, deviceModel := range deviceModels {
		devices := deviceModel.Devices
		for _, device := range devices {
			if device.Name == deviceName {
				conn, ok := p.connectors[device.ConnectionKey]
				if !ok {
					return nil, common.ConnectorNotFound
				}
				return conn, nil
			}
		}
	}
	return nil, common.ConnectorNotFound
}

// Destroy 销毁驱动
func (p *Plugin) Destroy() error {
	p.ls.Close()
	connectors := p.connectors
	for _, conn := range connectors {
		conn := conn
		go func() {
			conn.client.Disconnect(0)
		}()
	}
	return nil
}
