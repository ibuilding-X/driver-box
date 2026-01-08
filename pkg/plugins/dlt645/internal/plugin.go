package internal

import (
	"errors"
	"sync"
	"time"

	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/plugins/dlt645/internal/core/dltcon"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

const ProtocolName = "dlt645"

// Plugin 驱动插件
type Plugin struct {
	connPool map[string]*connector // 连接器
	ls       *lua.LState           // lua 虚拟机
	config   config.Config
}

// connector 连接器
type connector struct {
	config       *ConnectionConfig
	plugin       *Plugin
	client       *dltcon.Client
	keepAlive    bool      //串口保持打开状态
	latestIoTime time.Time // 最近一次执行IO的时间
	mutex        sync.Mutex
	retry        uint8 //通讯设备集合
	devices      map[string]*slaveDevice
	collectTask  *crontab.Future //当前连接的定时扫描任务
	close        bool            //当前连接是否已关闭
	virtual      bool            //是否虚拟链接
}

// Initialize 插件初始化
func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) {
	p.ls = ls
	p.config = c

	//初始化连接池
	p.initNetworks(c)
}

// 初始化Modbus连接池
func (p *Plugin) initNetworks(config config.Config) {
	p.connPool = make(map[string]*connector)
	//某个连接配置有问题，不影响其他连接的建立
	for key, connConfig := range config.Connections {
		// 按照串口采集
		connectionConfig := new(ConnectionConfig)
		if err := helper.Map2Struct(connConfig, connectionConfig); err != nil {
			helper.Logger.Error("convert connector config error", zap.Any("connection", connConfig), zap.Error(err))
			continue
		}
		// 打开串口
		conn, err := newConnector(p, connectionConfig)
		conn.config.ConnectionKey = key
		if err != nil {
			helper.Logger.Error("init dlt645 connector error", zap.Any("connection", connConfig), zap.Error(err))
			//continue
		}

		//生成点位采集组
		for _, model := range config.DeviceModels {
			for _, dev := range model.Devices {
				if dev.ConnectionKey != conn.config.ConnectionKey {
					continue
				}
				conn.createPointGroup(connectionConfig, model, dev)
			}
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
	_, ok = p.connPool[device.ConnectionKey]
	if !ok {
		helper.Logger.Error("not found connection key, key is ", zap.String("key", device.ConnectionKey), zap.Any("connections", p.connPool))
		return nil, errors.New("not found connection key, key is " + device.ConnectionKey)
	}
	return nil, nil
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
