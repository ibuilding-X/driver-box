package internal

import (
	"time"

	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
)

type primaryTable string

const BatchReadMode plugin.EncodeMode = "batchRead"
const (
	Coil            primaryTable = "COIL"             // 线圈
	DiscreteInput   primaryTable = "DISCRETE_INPUT"   // 离散输入
	InputRegister   primaryTable = "INPUT_REGISTER"   // 离散寄存器
	HoldingRegister primaryTable = "HOLDING_REGISTER" // 保持寄存器
)
const (
	ValueTypeBool         = "Bool"
	ValueTypeString       = "String"
	ValueTypeUint8        = "Uint8"
	ValueTypeUint16       = "Uint16"
	ValueTypeUint32       = "Uint32"
	ValueTypeUint64       = "Uint64"
	ValueTypeInt8         = "Int8"
	ValueTypeInt16        = "Int16"
	ValueTypeInt32        = "Int32"
	ValueTypeInt64        = "Int64"
	ValueTypeFloat32      = "Float32"
	ValueTypeFloat64      = "Float64"
	ValueTypeBinary       = "Binary"
	ValueTypeBoolArray    = "BoolArray"
	ValueTypeStringArray  = "StringArray"
	ValueTypeUint8Array   = "Uint8Array"
	ValueTypeUint16Array  = "Uint16Array"
	ValueTypeUint32Array  = "Uint32Array"
	ValueTypeUint64Array  = "Uint64Array"
	ValueTypeInt8Array    = "Int8Array"
	ValueTypeInt16Array   = "Int16Array"
	ValueTypeInt32Array   = "Int32Array"
	ValueTypeInt64Array   = "Int64Array"
	ValueTypeFloat32Array = "Float32Array"
	ValueTypeFloat64Array = "Float64Array"
	ValueTypeObject       = "Object"
)

// ConnectionConfig 连接器配置
type ConnectionConfig struct {
	plugin.BaseConnection
	Address       string `json:"address"`       // 地址：例如：127.0.0.1:502
	Mode          string `json:"mode"`          // 连接模式：rtuovertcp、rtu
	BaudRate      uint   `json:"baudRate"`      // 波特率（仅串口模式）
	DataBits      uint   `json:"dataBits"`      // 数据位（仅串口模式）
	StopBits      uint   `json:"stopBits"`      // 停止位（仅串口模式）
	Parity        uint   `json:"parity"`        // 奇偶性校验（仅串口模式）
	BatchReadLen  uint16 `json:"batchReadLen"`  // 最长连续读个数
	BatchWriteLen uint16 `json:"batchWriteLen"` // 支持连续写的最大长度
	MinInterval   uint16 `json:"minInterval"`   // 最小读取间隔
	Timeout       uint16 `json:"timeout"`       // 请求超时
	Retry         int    `json:"retry"`         // 重试次数
}

// Point modbus点位
type Point struct {
	config.Point
	//冗余设备相关信息
	DeviceId string

	//点位采集周期
	Duration     string `json:"duration"`
	Address      uint16
	RegisterType primaryTable `json:"primaryTable"`
	//该配置无需设置
	Quantity uint16 `json:"-"`
	Bit      uint8  `json:"bit"`
	BitLen   uint8  `json:"bitLen"`
	RawType  string `json:"rawType"`
	ByteSwap bool   `json:"byteSwap"`
	WordSwap bool   `json:"wordSwap"`
	//写操作是否强制要求多寄存器写接口。某些设备点位虽然只占据一个寄存器地址，但要求采用多寄存器写接口
	MultiWrite bool `json:"multiWrite"`
}

// 采集组
type slaveDevice struct {
	// 通讯设备，采集点位可以对应多个物模型设备
	unitID uint8
	//分组
	pointGroup []*pointGroup
}

type pointGroup struct {
	//分组索引
	index int
	// 从机地址
	UnitID uint8
	//采集间隔
	Duration time.Duration
	//寄存器类型
	RegisterType primaryTable
	//上一次采集时间
	LatestTime time.Time
	//记录最近连续超时次数
	TimeOutCount int
	//起始地址
	Address uint16
	//数量
	Quantity uint16
	Points   []*Point
}

// Connector#Send接入入参
type command struct {
	Mode  plugin.EncodeMode // 模式
	Value interface{}
}

// 写操作时 command的value类型
type writeValue struct {
	// 从机地址
	unitID  uint8
	Address uint16
	Value   []uint16
	//写操作是否强制要求多寄存器写接口
	MultiWrite   bool
	RegisterType primaryTable `json:"primaryTable"`
}
