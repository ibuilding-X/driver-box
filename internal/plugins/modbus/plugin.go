package modbus

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"strconv"
)

// Plugin 驱动插件
type Plugin struct {
	logger   *zap.Logger            // 日志记录器
	config   config.Config          // 核心配置
	adapter  plugin.ProtocolAdapter // 协议适配器
	connPool map[string]*connector  // 连接器
	ls       *lua.LState            // lua 虚拟机
}

// Initialize 插件初始化
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) (err error) {
	p.logger = logger
	p.config = c
	p.ls = ls

	// 初始化协议适配器
	p.adapter = &adapter{
		scriptDir: c.Key,
		ls:        ls,
	}
	p.connPool = make(map[string]*connector)

	for key, connConfig := range c.Connections {
		cc := new(connectorConfig)
		if err = helper.Map2Struct(connConfig, cc); err != nil {
			return fmt.Errorf("convert connector config error: %v", err)
		}
		c2, err := newConnector(p, cc)
		if err != nil {
			return fmt.Errorf("init connector error: %v", err)
		}
		p.connPool[key] = c2
	}

	for _, dm := range c.DeviceModels {
		for _, device := range dm.Devices {
			connectionKey := device.ConnectionKey
			conn, ok := p.connPool[connectionKey]
			if !ok {
				return fmt.Errorf("connection not found: %v", connectionKey)
			}
			uintID, ok := device.Properties["uintID"]
			if !ok {
				uintID = "1"
			}
			uintIdVal, err := strconv.ParseUint(uintID, 10, 8)
			if err != nil {
				return fmt.Errorf("convert slave id error: %v", err)
			}
			slaveId := uint8(uintIdVal)
			pointConfigMap, ok := conn.devices[slaveId]
			if !ok {
				pointConfigMap = map[primaryTable][]*pointConfig{
					Coil:            make([]*pointConfig, 0),
					DiscreteInput:   make([]*pointConfig, 0),
					InputRegister:   make([]*pointConfig, 0),
					HoldingRegister: make([]*pointConfig, 0),
				}
			}
			pointMap, ok := conn.pointMap[device.DeviceSn]
			if !ok {
				pointMap = make(map[string]*pointConfig)
			}

			for _, point := range dm.DevicePoints {
				tp := point.ToPoint()
				pc, err := convToPointConfig(tp.Extends)
				if err != nil {
					return fmt.Errorf("convToPointConfig error: %v", err)
				}
				pc.DeviceSn = device.DeviceSn
				pc.Name = tp.Name
				pc.ReadWrite = string(tp.ReadWrite)
				pc.SlaveId = slaveId
				pointConfigMap[pc.RegisterType] = append(pointConfigMap[pc.RegisterType], pc)
				pointMap[pc.Name] = pc
			}
			conn.devices[slaveId] = pointConfigMap
			conn.pointMap[device.DeviceSn] = pointMap
		}
	}

	for key, conn := range p.connPool {
		logger.Info(fmt.Sprintf("starting connection %s poll taskGroup", key))
		if err = conn.startPollTasks(); err != nil {
			return fmt.Errorf("start poll taskGroups error: %v", err)
		}
	}

	return nil
}

// ProtocolAdapter 适配器
func (p *Plugin) ProtocolAdapter() plugin.ProtocolAdapter {
	return p.adapter
}

// Connector 连接器
func (p *Plugin) Connector(deviceSn, pointName string) (conn plugin.Connector, err error) {
	// 获取连接key
	device, ok := helper.CoreCache.GetDeviceByDeviceAndPoint(deviceSn, pointName)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connPool[device.ConnectionKey]
	if !ok {
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	c.mutex.Lock()
	err = c.client.Open()
	if err != nil {
		c.mutex.Unlock()
		return nil, err
	}
	return c, nil
}

// Destroy 销毁驱动插件
func (p *Plugin) Destroy() error {
	for _, conn := range p.connPool {
		conn.polling = false
	}
	if p.ls != nil {
		p.ls.Close()
	}
	return nil
}
