package serial

import (
	"errors"
	"github.com/goburrow/serial"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"time"
)

type TimerGroup struct {
	//采集组唯一ID
	UUID       string
	LatestTime time.Time
	//记录最近连续超时次数
	TimeOutCount int
	//采集间隔
	Duration time.Duration
	//关联设备ID
	Devices []string
}
type ProtocolAdapter interface {
	//初始化采集任务组
	InitTimerGroup(connector *Connector) []TimerGroup
	//执行采集任务
	ExecuteTimerGroup(group *TimerGroup) error

	SendCommand(cmd Command) error

	DriverBoxEncode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res []Command, err error)
}

type SerialPort struct {
	client    serial.Port
	connector *Connector
}

// 只可在 plugin.Connector#Send方法中调用
func (s *SerialPort) Write(p []byte) (n int, err error) {
	s.connector.ensureInterval()
	return s.client.Write(p)
}

func (s *SerialPort) Read(p []byte) (n int, err error) {
	s.connector.ensureInterval()
	return s.client.Read(p)
}

func (s *SerialPort) close() {
	_ = s.client.Close()
}
func (c *Connector) ensureInterval() {
	np := c.latestIoTime.Add(time.Duration(c.Config.MinInterval) * time.Millisecond)
	if time.Now().Before(np) {
		time.Sleep(time.Until(np))
	}
	c.latestIoTime = time.Now()
}

// Connector 连接器
type Connector struct {
	plugin.Connection
	Config          *ConnectionConfig
	Plugin          *Plugin
	protocolAdapter ProtocolAdapter
	//当前串口的采集任务组
	TimerGroup   []TimerGroup
	Client       SerialPort
	timeout      time.Duration
	lastActivity time.Time
	t35          time.Duration
	t1           time.Duration
	//串口保持打开状态
	keepAlive    bool
	latestIoTime time.Time // 最近一次执行IO的时间
	mutex        sync.Mutex
	//通讯设备集合
	retry uint8

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
type Error string

// Error implements the error interface.
func (me Error) Error() (s string) {
	s = string(me)
	return
}

const (
	ErrRequestTimedOut Error = "request timed out"
)

type ConnectionConfig struct {
	plugin.BaseConnection
	Address     string `json:"address"`     // 地址：例如：127.0.0.1:502
	BaudRate    uint   `json:"baudRate"`    // 波特率（仅串口模式）
	DataBits    uint   `json:"dataBits"`    // 数据位（仅串口模式）
	StopBits    uint   `json:"stopBits"`    // 停止位（仅串口模式）
	Parity      string `json:"parity"`      // 奇偶性校验（仅串口模式）
	MinInterval uint16 `json:"minInterval"` // 最小读取间隔
	Timeout     uint16 `json:"timeout"`     // 请求超时
	Retry       int    `json:"retry"`       // 重试次数
}

type Command struct {
	Mode        plugin.EncodeMode // 模式
	MessageType string            //消息类型
	OutputFrame string            // 输出帧
	//串口响应回调
	Callback func(inputFrame []byte) error
}

func newConnector(p *Plugin, cf *ConnectionConfig) (*Connector, error) {
	if cf.MinInterval == 0 {
		cf.MinInterval = 100
	}
	if cf.Retry == 0 {
		cf.Retry = 3
	}
	if cf.Timeout <= 0 {
		cf.Timeout = 1000
	}

	conn := &Connector{
		Connection: plugin.Connection{},
		Config:     cf,
		Plugin:     p,
		virtual:    cf.Virtual || config.IsVirtual(),
	}
	conn.protocolAdapter = p.adapter
	return conn, nil
}

func (c *Connector) initCollectTask(conf *ConnectionConfig) (*crontab.Future, error) {
	if !conf.Enable {
		logger.Logger.Warn("modbus connection is disabled, ignore collect task", zap.String("key", c.ConnectionKey))
		return nil, nil
	}
	c.TimerGroup = c.Plugin.adapter.InitTimerGroup(c)
	//注册定时采集任务
	return helper.Crontab.AddFunc("1s", func() {
		if len(c.TimerGroup) == 0 {
			helper.Logger.Warn("no device to collect")
			return
		}
		for i, group := range c.TimerGroup {
			if c.close {
				helper.Logger.Warn("connection is closed, ignore collect task!", zap.String("key", c.ConnectionKey))
				break
			}
			duration := group.Duration
			if group.TimeOutCount > 0 {
				duration = duration * time.Duration(1<<group.TimeOutCount)
				//最大不超过一分钟
				if duration > time.Minute {
					duration = time.Minute
				} else if duration < 0 { //溢出，重置
					group.TimeOutCount = 0
					duration = group.Duration
				}
				helper.Logger.Warn("serial connection has timeout, increase duration", zap.Any("group", group), zap.Any("duration", duration))
			}
			//采集时间未到
			if group.LatestTime.Add(duration).After(time.Now()) {
				continue
			}

			//最近发生过写操作，推测当前时段可能存在其他设备的写入需求，采集任务主动避让
			if c.writeSemaphore.Load() > 0 || c.latestWriteTime.Add(time.Duration(conf.MinInterval)).After(time.Now()) {
				helper.Logger.Warn("modbus connection is writing, ignore collect task!", zap.String("key", c.ConnectionKey), zap.Any("semaphore", c.writeSemaphore.Load()))
				continue
			}

			helper.Logger.Debug("timer read modbus", zap.Any("group", i), zap.Any("latestTime", group.LatestTime), zap.Any("duration", group.Duration))
			//遍历所有通讯设备
			if err := c.Plugin.adapter.ExecuteTimerGroup(&group); err != nil {
				helper.Logger.Error("read error", zap.Any("connection", conf), zap.Any("group", group), zap.Error(err))
				//发生读超时，设备可能离线或者当前group点位配置有问题。将当前group的采集时间设置为未来值，跳过数个采集周期
				if errors.Is(err, ErrRequestTimedOut) {
					group.TimeOutCount += 1
				}
				//通讯失败，触发离线
				for _, deviceId := range group.Devices {
					_ = helper.DeviceShadow.MayBeOffline(deviceId)
				}
			} else {
				group.TimeOutCount = 0
			}
			group.LatestTime = time.Now()
		}

	})
}

// Decode 解码数据
func (c *Connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return nil, errors.New("请在 Command.CallBack 中调用 callback.ExportTo 以替换 callback.OnReceiveHandler 接口")
}

// Encode 编码数据
func (c *Connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	return c.protocolAdapter.DriverBoxEncode(deviceId, mode, values...)
}

// Send 发送数据
func (c *Connector) Send(data interface{}) error {
	err := c.openSerialPort()
	if err != nil {
		helper.Logger.Error("open serial port error", zap.Error(err))
		return err
	}
	cmd, ok := data.(Command)
	if ok {
		err = c.protocolAdapter.SendCommand(cmd)
		c.closeSerialPort(err)
		return err
	}
	cmds, ok := data.([]Command)
	if ok {
		for _, cmd = range cmds {
			err = c.protocolAdapter.SendCommand(cmd)
			if err != nil {
				c.closeSerialPort(err)
				return err
			}
		}
		c.closeSerialPort(err)
		return nil
	}
	return errors.New("unsupported data type")
}

// Release 释放资源
func (c *Connector) Release() (err error) {
	return nil
}

func (c *Connector) openSerialPort() error {
	c.mutex.Lock()
	//modbus连接已打开
	if c.keepAlive {
		return nil
	}
	var err error
	serialPort, err := serial.Open(&serial.Config{
		Address:  c.Config.Address,
		BaudRate: int(c.Config.BaudRate),
		DataBits: int(c.Config.DataBits),
		Parity:   c.Config.Parity,
		StopBits: int(c.Config.StopBits),
		Timeout:  time.Duration(c.Config.Timeout) * time.Millisecond,
	})
	helper.Logger.Info("serial config", zap.Any("serial", c.Config))
	if err != nil {
		c.mutex.Unlock()
		helper.Logger.Error("open serial port error", zap.Any("serial", c.Config), zap.Error(err))
	} else {
		c.Client = SerialPort{client: serialPort, connector: c}
		c.keepAlive = true
	}
	return err
}

func (c *Connector) closeSerialPort(e error) {
	defer func() {
		c.mutex.Unlock()
	}()
	if e != nil {
		helper.Logger.Error("serial error, will close it", zap.Error(e))
	}
	//RTU 模式下，连接不关闭
	if e != nil {
		c.keepAlive = false
		c.Client.close()
	}
}
func (c *Connector) Close() {
	c.close = true
	if c.collectTask != nil {
		c.collectTask.Disable()
	}
	if c.keepAlive {
		c.Client.close()
	}
}
