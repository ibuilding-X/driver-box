package plugin

import (
	"github.com/ibuilding-x/driver-box/pkg/event"
)

// ExportType 触发 ExportTo 的类型
// 定义了数据导出的不同触发类型
type ExportType string

// EncodeMode 编码模式
// 定义了数据编码的操作模式
type EncodeMode string

const (
	// ReadMode 读模式，用于从设备读取数据
	ReadMode EncodeMode = "read"

	// WriteMode 写模式，用于向设备写入数据
	WriteMode EncodeMode = "write"

	// RealTimeExport 实时上报类型，表示数据是实时变化上报的
	RealTimeExport ExportType = "realTimeExport"
)

// PointData 点位数据结构
// 表示单个设备点位的名称和值
type PointData struct {
	// PointName 点位名称，唯一标识设备上的一个测量点或控制点
	PointName string `json:"name"`

	// Value 点位值，可以是任意类型的数据，如数字、布尔值、字符串等
	Value interface{} `json:"value"`
}

// DeviceData 设备数据结构
// 表示单个设备的完整数据，包括点位值、事件和导出类型
type DeviceData struct {
	// ID 设备唯一标识符，用于区分不同的设备
	ID string `json:"id"`

	// Values 设备点位值数组，包含设备上所有有效点位的数据
	Values []PointData `json:"values"`

	// Events 设备相关事件数组，包含设备产生的各类事件
	Events []event.Data `json:"events"`

	// ExportType 上报类型，标识数据的来源类型
	// 底层的变化上报和实时上报等同于RealTimeExport
	ExportType ExportType
}

// PointReadValue 点位读操作的结构体
// 用于表示单个点位读取操作的结果
type PointReadValue struct {
	// ID 设备 ID
	ID string `json:"id"`

	// PointName 点位名称
	PointName string `json:"pointName"`

	// Value 点位值
	Value interface{} `json:"value"`
}

// BaseConnection 连接配置基础模型
// 定义了设备连接所需的基础配置信息
type BaseConnection struct {
	// ConnectionKey 连接标识符，用于唯一标识一个连接
	ConnectionKey string

	// ProtocolKey 协议驱动库标识，指定使用的通信协议
	ProtocolKey string `json:"protocolKey"`

	// Enable 是否启用此连接
	Enable bool `json:"enable"`

	//是否支持设备发现
	Discover bool `json:"discover"`

	// Virtual 是否为虚拟设备，虚拟设备不需要实际的物理连接
	Virtual bool `json:"virtual"`
}
