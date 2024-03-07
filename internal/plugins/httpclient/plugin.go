package httpclient

import (
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"net/http"
)

type Plugin struct {
	logger   *zap.Logger             // 日志记录器
	config   config.Config           // 核心配置
	callback plugin.OnReceiveHandler // 回调函数
	adapter  plugin.ProtocolAdapter  // 协议适配器
	connPool map[string]*connector   // 连接器
	ls       *lua.LState             // lua 虚拟机
}

func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, handler plugin.OnReceiveHandler, ls *lua.LState) (err error) {
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

func (p *Plugin) ProtocolAdapter() plugin.ProtocolAdapter {
	return p.adapter
}

// Connector 此协议不支持获取连接器
func (p *Plugin) Connector(deviceSn, pointName string) (connector plugin.Connector, err error) {
	// 获取连接key
	device, ok := helper.CoreCache.GetDeviceByDeviceAndPoint(deviceSn, pointName)
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
	if p.ls != nil {
		helper.Close(p.ls)
	}
	if len(p.connPool) > 0 {
		for i, _ := range p.connPool {
			if err := p.connPool[i].Release(); err != nil {
				return err
			}
		}
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
			client: &http.Client{},
		}
		p.connPool[key] = conn
	}
	return
}
