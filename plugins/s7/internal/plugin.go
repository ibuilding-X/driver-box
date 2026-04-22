package internal

import (
	"errors"
	"sync"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/ibuilding-x/driver-box/v2/pkg/convutil"
	"go.uber.org/zap"
)

const ProtocolName = "s7"

type Plugin struct {
	connPool map[string]*connector
	config   config.DeviceConfig
}

type connector struct {
	config      *ConnectionConfig
	plugin      *Plugin
	client      *s7Client
	mutex       sync.Mutex
	close       bool
	collectTask interface{}
	nodes       map[string]*NodeConfig
}

func (p *Plugin) Initialize(c config.DeviceConfig) {
	p.config = c
	p.initConnections(c)
}

func (p *Plugin) initConnections(config config.DeviceConfig) {
	p.connPool = make(map[string]*connector)
	for key, connConfig := range config.Connections {
		connectionConfig := new(ConnectionConfig)
		if err := convutil.Struct(connConfig, connectionConfig); err != nil {
			driverbox.Log().Error("convert connector config error", zap.Any("connection", connConfig), zap.Error(err))
			continue
		}
		conn, err := newConnector(p, connectionConfig)
		conn.config.ConnectionKey = key
		if err != nil {
			driverbox.Log().Error("init connector error", zap.Any("connection", connConfig), zap.Error(err))
			continue
		}
		for _, model := range config.DeviceModels {
			if len(model.Devices) == 0 {
				e := driverbox.CoreCache().DeleteModel(model.Name)
				if e != nil {
					driverbox.Log().Error("delete model error", zap.Any("model", model), zap.Error(e))
				} else {
					driverbox.Log().Warn("delete idle model", zap.Any("model", model))
				}
				continue
			}
			for _, dev := range model.Devices {
				if dev.ConnectionKey != conn.config.ConnectionKey {
					continue
				}
				conn.createNodeGroups(model, dev)
			}
		}
		if len(conn.nodes) == 0 {
			err = driverbox.CoreCache().DeleteConnection(conn.config.ConnectionKey)
			if err != nil {
				driverbox.Log().Error("delete connection error", zap.Any("connection", connConfig), zap.Error(err))
			} else {
				driverbox.Log().Warn("delete idle connection", zap.Any("connection", connConfig))
			}
			continue
		}
		if !connectionConfig.Enable {
			driverbox.Log().Warn("s7 connection is disabled", zap.String("key", key))
			continue
		}
		conn.collectTask, err = conn.initCollectTask(connectionConfig)
		p.connPool[key] = conn
		if err != nil {
			driverbox.Log().Error("init connector collect task error", zap.Any("connection", connConfig), zap.Error(err))
		}
	}
}

func (p *Plugin) Connector(deviceId string) (conn plugin.Connector, err error) {
	device, ok := driverbox.CoreCache().GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connPool[device.ConnectionKey]
	if !ok {
		driverbox.Log().Error("not found connection key", zap.String("key", device.ConnectionKey))
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	return c, nil
}

func (p *Plugin) Destroy() error {
	for _, conn := range p.connPool {
		conn.Close()
	}
	time.Sleep(time.Second * 1)
	return nil
}
