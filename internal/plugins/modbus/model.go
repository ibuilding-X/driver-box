package modbus

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/simonvetter/modbus"
	"sync"
	"time"
)

type primaryTable string

const (
	Coil            primaryTable = "COIL"             // 线圈
	DiscreteInput   primaryTable = "DISCRETE_INPUT"   // 离散输入
	InputRegister   primaryTable = "INPUT_REGISTER"   // 离散寄存器
	HoldingRegister primaryTable = "HOLDING_REGISTER" // 保持寄存器
)

// connectorConfig 连接器配置
type connectorConfig struct {
	Address     string `json:"address"`     // 地址：例如：127.0.0.1:502
	Mode        string `json:"mode"`        // 连接模式：rtuovertcp、rtu
	BaudRate    uint   `json:"baudRate"`    // 波特率（仅串口模式）
	DataBits    uint   `json:"dataBits"`    // 数据位（仅串口模式）
	StopBits    uint   `json:"stopBits"`    // 停止位（仅串口模式）
	Parity      uint   `json:"parity"`      // 奇偶性校验（仅串口模式）
	PollFreq    uint64 `json:"pollFreq"`    // 读取周期，
	MaxLen      uint16 `json:"maxLen"`      // 最长连续读个数
	MinInterval uint   `json:"minInterval"` // 最小读取间隔
	Timeout     int    `json:"timeout"`     // 请求超时
	Retry       int    `json:"retry"`       // 重试次数
	Duration    string `json:"duration"`    // 自动采集周期
	Virtual     bool   `json:"virtual"`     //虚拟设备功能
}

// PointExtend modbus点位扩展属性
type PointExtend struct {
	config.Point
	//冗余设备相关信息
	DeviceSn string

	//点位采集周期
	Duration     string `json:"duration"`
	SlaveId      uint8
	Address      uint16
	RegisterType primaryTable `json:"primaryTable"`
	Quantity     uint16       `json:"quantity"`
	Bit          int          `json:"bit"`
	BitLen       int          `json:"bitLen"`
	RawType      string       `json:"rawType"`
	ByteSwap     bool         `json:"byteSwap"`
	WordSwap     bool         `json:"wordSwap"`
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

// 采集组
type slaveDevice struct {
	// 通讯设备，采集点位可以对应多个物模型设备
	unitID uint8
	//分组
	pointGroup []*pointGroup
}

type pointGroup struct {
	// 从机地址
	unitID uint8
	//采集间隔
	Duration time.Duration
	//寄存器类型
	RegisterType primaryTable
	//上一次采集时间
	LatestTime time.Time
	//起始地址
	Address uint16
	//数量
	Quantity uint16
	points   []*PointExtend
}

// Connector#Send接入入参
type command struct {
	mode  plugin.EncodeMode // 模式
	value interface{}
}

// 写操作时 command的value类型
type writeValue struct {
	// 从机地址
	unitID       uint8
	Address      uint16
	Value        interface{}
	Quantity     uint16       `json:"quantity"`
	RegisterType primaryTable `json:"primaryTable"`
	Bit          int          `json:"bit"`
	BitLen       int          `json:"bitLen"`
	RawType      string       `json:"rawType"`
	ByteSwap     bool         `json:"byteSwap"`
	WordSwap     bool         `json:"wordSwap"`
}
