package internal

import (
	"errors"
	"net/http"

	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

const ProtocolName = "http_client"

type Plugin struct {
	logger   *zap.Logger           // 日志记录器
	config   config.Config         // 核心配置
	connPool map[string]*connector // 连接器
	ls       *lua.LState           // lua 虚拟机
}

func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) {
	p.logger = logger
	p.config = c
	p.ls = ls

	// 初始化连接池
	if err := p.initConnPool(); err != nil {
		logger.Error("init connector pool failed", zap.Error(err))
	}

}

// Connector 此协议不支持获取连接器
func (p *Plugin) Connector(deviceSn string) (connector plugin.Connector, err error) {
	// 获取连接key
	device, ok := helper.CoreCache.GetDevice(deviceSn)
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

func (p *Plugin) initConnPool() (err error) {
	p.connPool = make(map[string]*connector)
	for key, _ := range p.config.Connections {
		var c connectorConfig
		if err = helper.Map2Struct(p.config.Connections[key], &c); err != nil {
			return
		}
		if c.Timeout <= 0 {
			c.Timeout = 5000
		}
		c.ConnectionKey = key
		conn := &connector{
			plugin: p,
			config: c,
			client: &http.Client{},
		}
		conn.initCollectTask()
		p.connPool[key] = conn
	}
	return
}
