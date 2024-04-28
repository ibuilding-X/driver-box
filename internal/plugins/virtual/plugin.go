package virtual

//
//import (
//	"github.com/ibuilding-x/driver-box/driverbox/config"
//	"github.com/ibuilding-x/driver-box/driverbox/plugin"
//	lua "github.com/yuin/gopher-lua"
//	"go.uber.org/zap"
//)
//
//type Plugin struct {
//	logger    *zap.Logger
//	config    config.Config
//	adapter   *adapter
//	connector plugin.Connector
//	ls        *lua.LState
//}
//
//func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) (err error) {
//	p.logger = logger
//	p.config = c
//	p.ls = ls
//
//	// 初始化适配器
//	p.adapter = &adapter{
//		scriptDir: c.Key,
//		ls:        ls,
//	}
//
//	// 初始化连接
//	p.connector = &connector{
//		plugin: p,
//	}
//
//	return nil
//}
//
//func (p *Plugin) ProtocolAdapter() plugin.ProtocolAdapter {
//	return p.adapter
//}
//
//func (p *Plugin) Connector(deviceSn, pointName string) (connector plugin.Connector, err error) {
//	return p.connector, nil
//}
//
//func (p *Plugin) Destroy() error {
//	return nil
//}
