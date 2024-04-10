package modbus

import (
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/simonvetter/modbus"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"sync"
	"time"
)

// Plugin 驱动插件
type Plugin struct {
	adapter  plugin.ProtocolAdapter // 协议适配器
	connPool map[string]*connector  // 连接器
	ls       *lua.LState            // lua 虚拟机
}

// connector 连接器
type connector struct {
	key          string
	plugin       *Plugin
	client       *modbus.ModbusClient
	maxLen       uint16    // 最长连续读个数
	minInterval  uint      // 读取间隔
	polling      bool      // 执行轮询
	lastPoll     time.Time // 上次轮询
	latestIoTime time.Time // 最近一次执行IO的时间
	mutex        sync.Mutex
	//通讯设备集合
	retry int

	devices map[string]*slaveDevice
	//当前连接的定时扫描任务
	collectTask *crontab.Future
	//当前连接是否已关闭
	close bool
	//是否虚拟链接
	virtual bool
}

// Initialize 插件初始化
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) (err error) {
	p.ls = ls

	// 初始化协议适配器
	p.adapter = &adapter{
		ls:           ls,
		scriptEnable: helper.ScriptExists(c.Key),
	}
	//初始化连接池
	p.initNetworks(c)
	return nil
}

// 初始化Modbus连接池
func (p *Plugin) initNetworks(config config.Config) {
	p.connPool = make(map[string]*connector)
	//某个连接配置有问题，不影响其他连接的建立
	for key, connConfig := range config.Connections {
		connectionConfig := new(ConnectionConfig)
		if err := helper.Map2Struct(connConfig, connectionConfig); err != nil {
			helper.Logger.Error("convert connector config error", zap.Any("connection", connConfig), zap.Error(err))
			continue
		}
		conn, err := newConnector(p, connectionConfig)
		if err != nil {
			helper.Logger.Error("init connector error", zap.Any("connection", connConfig), zap.Error(err))
			continue
		}

		//生成点位采集组
		for _, model := range config.DeviceModels {
			for _, dev := range model.Devices {
				if dev.ConnectionKey != conn.key {
					continue
				}
				conn.createPointGroup(model, dev)
			}
		}

		//启动采集任务
		duration := connectionConfig.Duration
		if duration == "" {
			helper.Logger.Warn("modbus connection duration is empty, use default 5s", zap.String("key", conn.key))
			duration = "5s"
		}
		conn.collectTask, err = conn.initCollectTask(duration)
		p.connPool[key] = conn
		if err != nil {
			helper.Logger.Error("init connector collect task error", zap.Any("connection", connConfig), zap.Error(err))
		}
	}
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
