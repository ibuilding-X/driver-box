package modbus

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"time"
)

type primaryTable string

const (
	Coil            primaryTable = "COIL"             // 线圈
	DiscreteInput   primaryTable = "DISCRETE_INPUT"   // 离散输入
	InputRegister   primaryTable = "INPUT_REGISTER"   // 离散寄存器
	HoldingRegister primaryTable = "HOLDING_REGISTER" // 保持寄存器
)

// ConnectionConfig 连接器配置
type ConnectionConfig struct {
	Address     string `json:"address"`     // 地址：例如：127.0.0.1:502
	Mode        string `json:"mode"`        // 连接模式：rtuovertcp、rtu
	BaudRate    uint   `json:"baudRate"`    // 波特率（仅串口模式）
	DataBits    uint   `json:"dataBits"`    // 数据位（仅串口模式）
	StopBits    uint   `json:"stopBits"`    // 停止位（仅串口模式）
	Parity      uint   `json:"parity"`      // 奇偶性校验（仅串口模式）
	MaxLen      uint16 `json:"maxLen"`      // 最长连续读个数
	MinInterval uint   `json:"minInterval"` // 最小读取间隔
	Timeout     int    `json:"timeout"`     // 请求超时
	Retry       int    `json:"retry"`       // 重试次数
	Duration    string `json:"duration"`    // 自动采集周期
	Virtual     bool   `json:"virtual"`     //虚拟设备功能
}

// Point modbus点位
type Point struct {
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
	points   []*Point
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
