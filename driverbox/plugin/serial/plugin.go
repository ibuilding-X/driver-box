package serial

import (
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

const ProtocolName = "serial"

// Plugin 驱动插件
type Plugin struct {
	connPool map[string]*Connector // 连接器
	ls       *lua.LState           // lua 虚拟机
	Config   config.Config
	//具体协议的采集任务实现
	adapter ProtocolAdapter
}

func NewPlugin(adapter ProtocolAdapter) *Plugin {
	return &Plugin{adapter: adapter}
}

// Initialize 插件初始化
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) {
	p.ls = ls
	p.Config = c
	//初始化连接池
	p.connPool = make(map[string]*Connector)
	//某个连接配置有问题，不影响其他连接的建立
	for key, connConfig := range c.Connections {
		connectionConfig := new(ConnectionConfig)
		if err := helper.Map2Struct(connConfig, connectionConfig); err != nil {
			helper.Logger.Error("convert connector config error", zap.Any("connection", connConfig), zap.Error(err))
			continue
		}
		conn, err := newConnector(p, connectionConfig)
		conn.Config.ConnectionKey = key
		if err != nil {
			helper.Logger.Error("init connector error", zap.Any("connection", connConfig), zap.Error(err))
			continue
		}

		//启动采集任务
		conn.collectTask, err = conn.initCollectTask(connectionConfig)
		p.connPool[key] = conn
		if err != nil {
			helper.Logger.Error("init connector collect task error", zap.Any("connection", connConfig), zap.Error(err))
		}
	}
}

// Connector 连接器
func (p *Plugin) Connector(deviceId string) (conn plugin.Connector, err error) {
	// 获取连接key
	device, ok := helper.CoreCache.GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connPool[device.ConnectionKey]
	if !ok {
		helper.Logger.Error("not found connection key, key is ", zap.String("key", device.ConnectionKey), zap.Any("connections", p.connPool))
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	return c, nil
}

// Destroy 销毁驱动插件
func (p *Plugin) Destroy() error {
	for _, conn := range p.connPool {
		conn.Close()
	}
	if p.ls != nil {
		helper.Close(p.ls)
	}
	return nil
}
