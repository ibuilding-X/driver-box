package internal

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"github.com/ibuilding-x/driver-box/pkg/crontab"
	"github.com/ibuilding-x/driver-box/pkg/luautil"
	"github.com/simonvetter/modbus"
	"go.uber.org/zap"
)

const ProtocolName = "modbus"

// Plugin 驱动插件
type Plugin struct {
	connPool map[string]*connector // 连接器
	config   config.DeviceConfig
}

// connector 连接器
type connector struct {
	config *ConnectionConfig
	plugin *Plugin
	client *modbus.ModbusClient
	//串口保持打开状态
	keepAlive    bool
	latestIoTime time.Time // 最近一次执行IO的时间
	mutex        sync.Mutex
	//通讯设备集合
	retry uint8

	devices map[uint8]*slaveDevice
	//当前连接的定时扫描任务
	collectTask *crontab.Future
	//当前连接是否已关闭
	close bool
	//是否虚拟链接
	virtual bool

	//写操作信号量
	writeSemaphore  atomic.Int32
	latestWriteTime time.Time //最近一次写操作时间

	writeEncodeMu sync.Mutex
}

// Initialize 插件初始化
func (p *Plugin) Initialize(c config.DeviceConfig) {
	p.config = c
	//初始化连接池
	p.initNetworks(c)

}

// 初始化Modbus连接池
func (p *Plugin) initNetworks(config config.DeviceConfig) {
	p.connPool = make(map[string]*connector)
	//某个连接配置有问题，不影响其他连接的建立
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
		if conn.virtual {
			InitMockLua()
		}

		//生成点位采集组
		for _, model := range config.DeviceModels {
			//如果模型不存在关联设备,清理该模型
			if len(model.Devices) == 0 {
				e := driverbox.CoreCache().DeleteModel(model.Name)
				driverbox.Log().Warn("delete model error", zap.Any("model", model), zap.Error(e))
				continue
			}
			for _, dev := range model.Devices {
				if dev.ConnectionKey != conn.config.ConnectionKey {
					continue
				}
				conn.createPointGroup(connectionConfig, model, dev)
			}
		}

		if len(conn.devices) == 0 {
			err = driverbox.CoreCache().DeleteConnection(conn.config.ConnectionKey)
			driverbox.Log().Warn("modbus connection has no device to collect,remove it", zap.String("key", key), zap.Error(err))
			continue
		}
		if !connectionConfig.Enable {
			driverbox.Log().Warn("modbus connection is disabled, ignore collect task", zap.String("key", key))
			continue
		}

		//启动采集任务
		conn.collectTask, err = conn.initCollectTask(connectionConfig)
		p.connPool[key] = conn
		if err != nil {
			driverbox.Log().Error("init connector collect task error", zap.Any("connection", connConfig), zap.Error(err))
		}
	}
}

// Connector 连接器
func (p *Plugin) Connector(deviceId string) (conn plugin.Connector, err error) {
	// 获取连接key
	device, ok := driverbox.CoreCache().GetDevice(deviceId)
	if !ok {
		return nil, errors.New("not found device connection key")
	}
	c, ok := p.connPool[device.ConnectionKey]
	if !ok {
		driverbox.Log().Error("not found connection key, key is ", zap.String("key", device.ConnectionKey), zap.Any("connections", p.connPool))
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	return c, nil
}

// Destroy 销毁驱动插件
func (p *Plugin) Destroy() error {
	for _, conn := range p.connPool {
		conn.Close()
	}
	if ls != nil {
		luautil.Close(ls)
		ls = nil
	}
	//延迟关闭lua虚拟机，防止lua虚拟机正在使用
	time.Sleep(time.Second * 1)
	return nil
}
