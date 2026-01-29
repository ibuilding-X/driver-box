package plugin

import (
	"github.com/ibuilding-x/driver-box/v2/pkg/event"
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
// 表示单个设备点位的名称和值，用于在系统中传递点位信息
type PointData struct {
	// PointName 点位名称，唯一标识设备上的一个测量点或控制点
	// 必须与设备模型中定义的点位名称一致
	PointName string `json:"name"`

	// Value 点位值，可以是任意类型的数据，如数字、布尔值、字符串等
	// 值的类型应与点位定义的ValueType匹配
	Value interface{} `json:"value"`
}

// DeviceData 设备数据结构
// 表示单个设备的完整数据，包括点位值、事件和导出类型
// 该结构用于在插件、导出模块和核心服务之间传递设备数据
type DeviceData struct {
	// ID 设备唯一标识符，用于区分不同的设备
	// 必须与系统中注册的设备ID一致
	ID string `json:"id"`

	// Values 设备点位值数组，包含设备上所有有效点位的数据
	// 每个元素为PointData结构，包含点位名称和值
	Values []PointData `json:"values"`

	// Events 设备相关事件数组，包含设备产生的各类事件
	// 事件可能包括设备状态变化、告警、连接状态等
	Events []event.Data `json:"events"`

	// ExportType 上报类型，标识数据的来源类型
	// 底层的变化上报和实时上报等同于RealTimeExport
	ExportType ExportType
}

// PointReadValue 点位读操作的结构体
// 用于表示单个点位读取操作的结果，通常在读取响应中使用
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
// 该结构通常作为更具体的连接配置的基类使用
type BaseConnection struct {
	// ConnectionKey 连接标识符，用于唯一标识一个连接
	// 必须全局唯一，用于多设备连接管理
	ConnectionKey string

	// ProtocolKey 协议驱动库标识，指定使用的通信协议
	// 如 "modbus", "bacnet", "mqtt" 等
	ProtocolKey string `json:"protocolKey"`

	// Enable 是否启用此连接
	// false表示禁用该连接，不会尝试建立通信
	Enable bool `json:"enable"`

	// Discover 是否支持设备发现
	// true表示该连接支持自动发现设备功能
	Discover bool `json:"discover"`

	// Virtual 是否为虚拟设备，虚拟设备不需要实际的物理连接
	// true表示设备为虚拟设备，常用于测试或模拟场景
	Virtual bool `json:"virtual"`
}
