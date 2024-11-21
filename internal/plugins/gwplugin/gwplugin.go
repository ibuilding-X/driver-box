package gwplugin

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

type gatewayPlugin struct {
	l  *zap.Logger
	c  config.Config
	ls *lua.LState
}

func (g *gatewayPlugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) {
	g.l = logger
	g.c = c
	g.ls = ls
}

func (g *gatewayPlugin) Connector(deviceId string) (connector plugin.Connector, err error) {
	//TODO implement me
	panic("implement me")
}

func (g *gatewayPlugin) Destroy() error {
	//TODO implement me
	panic("implement me")
}

func New() plugin.Plugin {
	return &gatewayPlugin{}
}
