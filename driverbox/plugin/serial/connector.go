package serial

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/goburrow/serial"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/library"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/logger"
	glua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type TimerGroup struct {
	LatestTime time.Time
	//记录最近连续超时次数
	TimeOutCount int
	//采集间隔
	Duration time.Duration
	//关联设备ID
	Devices []string
}
type TimerTask interface {
	Init(config.Config) []TimerGroup
	EncodeToCommand(group *TimerGroup) *Command
}

// connector 连接器
type connector struct {
	plugin.Connection
	config *ConnectionConfig
	plugin *Plugin
	//当前串口的采集任务组
	TimerGroup   []TimerGroup
	client       serial.Port
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
	inputFrame  []byte            // 输入帧
}

func newConnector(p *Plugin, cf *ConnectionConfig) (*connector, error) {
	if cf.MinInterval == 0 {
		cf.MinInterval = 100
	}
	if cf.Retry == 0 {
		cf.Retry = 3
	}
	if cf.Timeout <= 0 {
		cf.Timeout = 1000
	}

	conn := &connector{
		Connection: plugin.Connection{},
		config:     cf,
		plugin:     p,
		virtual:    cf.Virtual || config.IsVirtual(),
	}

	return conn, nil
}

func (c *connector) initCollectTask(conf *ConnectionConfig) (*crontab.Future, error) {
	if !conf.Enable {
		logger.Logger.Warn("modbus connection is disabled, ignore collect task", zap.String("key", c.ConnectionKey))
		return nil, nil
	}
	c.TimerGroup = c.plugin.timerTask.Init(c.plugin.config)
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
			cmd := c.plugin.timerTask.EncodeToCommand(&group)
			if err := c.Send(cmd); err != nil {
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
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	cmd := raw.(*Command)
	return library.Protocol().DecodeV2(c.config.ProtocolKey, func(L *glua.LState) *glua.LTable {
		table := L.NewTable()
		table.RawSetString("inputFrame", glua.LString(cmd.inputFrame))
		table.RawSetString("messageType", glua.LString(cmd.MessageType))
		table.RawSetString("outputFrame", glua.LString(cmd.OutputFrame))
		if cmd.Mode == plugin.WriteMode {
			table.RawSetString("mode", glua.LString("write"))
		} else {
			table.RawSetString("mode", glua.LString("read"))
		}
		return table
	})
}

// Encode 编码数据
func (c *connector) Encode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	_, err = library.Protocol().EncodeV2(c.config.ProtocolKey, library.ProtocolEncodeRequest{
		DeviceId: deviceId,
		Mode:     mode,
		Points:   values,
	})

	return nil, err
}

// Send 发送数据
func (c *connector) Send(data interface{}) error {
	cmd := data.(*Command)

	output := cmd.OutputFrame
	//移除空格
	output = strings.ReplaceAll(output, " ", "")

	bytes, err := hex.DecodeString(output)
	if err != nil {
		helper.Logger.Error("hex decode error", zap.Error(err))
		return err
	}
	err = c.openSerialPort()
	if err != nil {
		return err
	}
	defer c.closeSerialPort(err)

	var ts time.Time
	var t time.Duration
	var n int
	t = time.Since(c.lastActivity.Add(c.t35))
	if t < 0 {
		time.Sleep(t * (-1))
	}

	ts = time.Now()
	fmt.Println("write data:" + hex.Dump(bytes))
	n, err = c.client.Write(bytes)
	if err != nil {
		return err
	}
	c.lastActivity = ts.Add(time.Duration(n) * c.t1)

	// observe inter-frame delays
	time.Sleep(c.lastActivity.Add(c.t35).Sub(time.Now()))
	rxbuf := make([]byte, 256)
	byteCount, err := io.ReadFull(c.client, rxbuf)
	cmd.inputFrame = rxbuf[:byteCount]
	fmt.Println("read response:" + hex.EncodeToString(cmd.inputFrame))

	_, err = callback.OnReceiveHandler(c, cmd)
	return err
}

// Release 释放资源
// 不释放连接资源，经测试该包不支持频繁创建连接
func (c *connector) Release() (err error) {
	return
}

// ensureInterval 确保与前一次IO至少间隔minInterval毫秒
func (c *connector) ensureInterval() {
	np := c.latestIoTime.Add(time.Duration(c.config.MinInterval) * time.Millisecond)
	if time.Now().Before(np) {
		time.Sleep(time.Until(np))
	}
	c.latestIoTime = time.Now()
}

func (c *connector) openSerialPort() error {
	c.mutex.Lock()
	//modbus连接已打开
	if c.keepAlive {
		return nil
	}
	var err error
	c.client, err = serial.Open(&serial.Config{
		Address:  c.config.Address,
		BaudRate: int(c.config.BaudRate),
		DataBits: int(c.config.DataBits),
		Parity:   c.config.Parity,
		StopBits: int(c.config.StopBits),
		Timeout:  10 * time.Millisecond,
	})
	if err != nil {
		c.mutex.Unlock()
		helper.Logger.Error("open serial port error", zap.Any("serial", c.config), zap.Error(err))
	} else {
		c.keepAlive = true
	}
	return err
}

func (c *connector) closeSerialPort(e error) {
	defer func() {
		c.mutex.Unlock()
	}()
	if e != nil {
		helper.Logger.Error("modbus client error, will close it", zap.Error(e))
	}
	//RTU 模式下，连接不关闭
	if e != nil {
		c.keepAlive = false
		_ = c.client.Close()
	}
}
func (c *connector) Close() {
	c.close = true
	if c.collectTask != nil {
		c.collectTask.Disable()
	}
	if c.keepAlive {
		_ = c.client.Close()
	}
}
