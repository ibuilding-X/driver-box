package dto

// ValueType 数据类型：int、float、string
type ValueType string

// ReadWrite 读写模式：R、W、RW
type ReadWrite string

// ReportMode 上报模式：realTime、change
type ReportMode string

const (
	ValueTypeInt    ValueType = "int"
	ValueTypeFloat  ValueType = "float"
	ValueTypeString ValueType = "string"
)

const (
	ReadWriteR  ReadWrite = "R"
	ReadWriteW  ReadWrite = "W"
	ReadWriteRW ReadWrite = "RW"
)

const (
	ReportModeReal   ReportMode = "realTime"
	ReportModeChange ReportMode = "change"
)

type Config struct {
	// 模型列表（Model.Name 作为 key）
	Models map[string]Model `json:"models"`
	// 连接参数
	Connections map[string]H `json:"connections"`
	// 协议名称
	ProtocolName string `json:"protocolName"`
}

type Model struct {
	// 模型唯一标识
	Name string `json:"name"`
	// 模型 ID
	ModelId string `json:"modelId"`
	// 模型描述
	Description string `json:"description"`
	// 扩展属性
	Attributes H `json:"attributes"`
	// 点位列表
	Points map[string]Point `json:"points"`
	// 设备列表
	Devices map[string]Device `json:"devices"`
	// 协议名称
	ProtocolName string `json:"protocolName"`
}

type Point struct {
	// 点位名称
	Name string `json:"name"`
	// 点位描述
	Description string `json:"description"`
	// 值类型
	ValueType ValueType `json:"ValueType"`
	// 读写模式
	ReadWrite ReadWrite `json:"readWrite"`
	// 上报模式
	ReportMode ReportMode `json:"reportMode"`
	// 单位
	Units string `json:"units"`
	//数值精度
	Scale float64 `json:"scale"`
	//保留小数位数
	Decimals int `json:"decimals"`
	// 点位枚举表
	Enums []PointEnum `json:"enums"`
	// 扩展参数
	Extends H `json:"extends"`
}

type PointEnum struct {
	//枚举名称
	Name string `json:"name"`
	//枚举值
	Value interface{} `json:"value"`
	//枚举图标：用于界面展示
	Icon string `json:"icon"`
}

type Device struct {
	// 设备唯一标识
	Id string `json:"id"`
	// 模型名称
	ModelName string `json:"modelName"`
	// 模型 ID
	ModelId string `json:"modelId"`
	// 设备描述
	Description string `json:"description"`
	// 设备离线阈值，超过该时长没有收到数据视为离线。例如："15m"
	Ttl string `json:"ttl"`
	// 设备标签（暂未使用）
	Tags []string `json:"tags"`
	// 连接 Key
	ConnectionKey string `json:"connectionKey"`
	// 协议参数
	Properties HS `json:"properties"`
	// 设备层驱动的引用
	DriverKey string `json:"driverKey"`
	// 协议名称
	ProtocolName string `json:"protocolName"`
}
