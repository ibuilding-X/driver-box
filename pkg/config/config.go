// Package config 核心配置
package config

import (
	"encoding/json"
	"os"

	"github.com/go-playground/validator/v10"
)

// 环境变量配置项
const (
	//资源文件存放目录
	ENV_RESOURCE_PATH = "DRIVERBOX_RESOURCE_PATH"
	//http服务监听端口号
	ENV_HTTP_LISTEN = "DRIVERBOX_HTTP_LISTEN"

	//UDP discover服务监听端口号
	ENV_UDP_DISCOVER_LISTEN = "DRIVERBOX_UDP_DISCOVER_LISTEN"

	//日志文件存放路径
	ENV_LOG_PATH = "DRIVERBOX_LOG_PATH"

	//是否虚拟设备模式: true:是,false:否
	ENV_VIRTUAL = "DRIVERBOX_VIRTUAL"

	//是否虚拟设备模式: true:是,false:否
	ENV_LUA_PRINT_ENABLED = "DRIVERBOX_LUA_PRINT_ENABLE"

	//场景联动配置存放目录
	ENV_LINKEDGE_CONFIG_PATH = "EXPORT_LINKEDGE_CONFIG_PATH"

	//镜像设备功能是否可用
	ENV_EXPORT_MIRROR_ENABLED = "EXPORT_MIRROR_ENABLED"

	//镜像设备功能是否可用
	ENV_EXPORT_DISCOVER_ENABLED = "EXPORT_DISCOVER_ENABLED"

	//driver-box默认UI是否可用
	ENV_EXPORT_UI_ENABLED = "EXPORT_UI_ENABLED"

	//driver-box llm-agent是否可用
	ENV_EXPORT_LLM_AGENT_ENABLED = "EXPORT_LLM_AGENT_ENABLED"

	//MCP功能是否可用
	ENV_EXPORT_MCP_ENABLED = "EXPORT_MCP_ENABLED"

	//设备历史数据存放路径
	EXPORT_HISTORY_DATA_PATH = "EXPORT_HISTORY_DATA_PATH"
	//设备历史数据保存时长，单位（天），默认值：14
	EXPORT_HISTORY_RESERVED_DAYS = "EXPORT_HISTORY_RESERVED_DAYS"
	//剖面数据写入频率，默认值：60s
	EXPORT_HISTORY_SNAPSHOT_FLUSH_INTERVAL = "EXPORT_HISTORY_SNAPSHOT_FLUSH_INTERVAL"
	//实时数据写入频率，默认值：5s
	EXPORT_HISTORY_REAL_TIME_FLUSH_INTERVAL = "EXPORT_HISTORY_REAL_TIME_FLUSH_INTERVAL"
)

// 点位上报模式
type ReportMode string

// 点位读写模式
type ReadWrite string

// 点位数据类型
type ValueType string

var (
	//实时上报,读到数据即触发
	ReportMode_Real ReportMode = "realTime"
	//变化上报,同影子中数值不一致时才触发上报
	ReportMode_Change ReportMode = "change"
	//只读
	ReadWrite_R ReadWrite = "R"
	//只写
	ReadWrite_W ReadWrite = "W"
	//读写
	ReadWrite_RW ReadWrite = "RW"
	//点位类型：整型
	ValueType_Int ValueType = "int"
	//点位类型：浮点型
	ValueType_Float ValueType = "float"
	//点位类型：字符串
	ValueType_String ValueType = "string"
)

// 资源文件目录
var ResourcePath = "./res"

type EnvConfig struct {
	ConfigPath string
	//http服务监听端口号
	HttpListen string
	LogPath    string
}

type Point map[string]interface{} // 点位 Map，可转换为标准点位数据

// Name 获取点位名称
// 返回点位的名称，该名称是点位的唯一标识符
func (pm Point) Name() string {
	return pm["name"].(string)
}

// ReadWrite 获取点位读写模式
// 返回点位的读写权限设置，如只读、只写或读写
func (pm Point) ReadWrite() ReadWrite {
	valueType, ok := pm.FieldValue("readWrite")
	if !ok {
		return ""
	}
	return ReadWrite(valueType.(string))
}

// FieldValue 根据键名获取点位字段值
// 参数 key: 字段键名
// 返回值 v: 字段值, exists: 字段是否存在
func (pm Point) FieldValue(key string) (v interface{}, exists bool) {
	v, exists = pm[key]
	return
}

// Description 获取点位描述信息
// 返回点位的详细描述文本，用于说明点位的用途和含义
func (pm Point) Description() string {
	return pm["description"].(string)
}

// Enums 获取点位枚举值列表
// 返回点位支持的枚举值数组，如果未设置则返回空数组
func (p Point) Enums() []PointEnum {
	enums := make([]PointEnum, 0)
	v, ok := p.FieldValue("enums")
	if !ok {
		return enums
	}
	b, err := json.Marshal(v)
	if err == nil {
		json.Unmarshal(b, &enums)
	}
	return enums
}

// ValueType 获取点位数据类型
// 返回点位的数据类型，如整型、浮点型、布尔型等
func (pm Point) ValueType() ValueType {
	valueType, ok := pm["valueType"]
	if !ok {
		return ""
	}
	return ValueType(valueType.(string))
}

// ReportMode 获取点位上报模式
// 返回点位的数据上报模式，如实时上报、变化上报等
// 如果配置中未指定，则默认为实时上报模式
func (pm Point) ReportMode() ReportMode {
	reportMode, ok := pm.FieldValue("reportMode")
	if !ok {
		return ReportMode_Real
	}
	return ReportMode(reportMode.(string))
}

// Scale 获取点位缩放比例
// 返回点位数值的缩放系数，用于数值转换，默认为0（无缩放）
func (pm Point) Scale() float64 {
	scale, ok := pm["scale"]
	if !ok {
		return 0
	}
	return scale.(float64)
}

// Decimals 获取点位小数位数
// 返回点位数值保留的小数位数
// 对于浮点数类型，默认保留2位小数；对于其他类型，默认为0位小数
func (pm Point) Decimals() int {
	decimals, ok := pm["decimals"]
	if !ok {
		//浮点数，且未指定decimals，默认未2
		if pm.ValueType() == ValueType_Float {
			return 2
		} else {
			return 0
		}
	}
	switch decimals.(type) {
	case float64:
		return int(decimals.(float64))
	default:
		return decimals.(int)
	}
}

// Units 获取点位单位
// 返回点位数值的单位标识，如℃、kW、m³等
func (pm Point) Units() string {
	defaultValue, ok := pm.FieldValue("units")
	if !ok {
		return ""
	}
	return defaultValue.(string)
}

// Config 配置
type Config struct {
	// 设备模型
	DeviceModels []DeviceModel `json:"deviceModels" validate:""`
	// 连接配置
	Connections map[string]interface{} `json:"connections" validate:""`
	// 协议名称（通过协议名称区分连接模式：客户端、服务端）
	PluginName string `json:"protocolName" validate:"required"`
}

//------------------------------ 设备模型 ------------------------------

// Model 模型基础信息
type Model struct {
	// 模型名称
	Name string `json:"name" validate:"required"`
	// 云端模型 ID
	ModelID string `json:"modelId" validate:"required"`
	// 模型描述
	Description string `json:"description" validate:"required"`
	//扩展属性
	Attributes map[string]interface{} `json:"attributes"`
	// 模型点位列表
	DevicePoints []Point `json:"devicePoints" validate:"required"`
}

// DeviceModel 设备模型
type DeviceModel struct {
	Model
	// 设备列表
	Devices []Device `json:"devices" validate:"required"`
}

type PointEnum struct {
	//枚举名称
	Name string `json:"name"`
	//枚举值
	Value interface{} `json:"value"`
	//枚举图标：用于界面展示
	Icon string `json:"icon"`
}

//------------------------------ 设备 ------------------------------

// Device 设备
type Device struct {
	// 设备 ID
	ID string `json:"id" validate:"required"`
	// 模型名称
	ModelName string `json:"-" validate:"-"`
	// 设备描述
	Description string `json:"description" validate:"required"`
	// 设备离线阈值，超过该时长没有收到数据视为离线
	Ttl string `json:"ttl"`

	//设备标签
	Tags []string `json:"tags"`
	// 连接 Key
	ConnectionKey string `json:"connectionKey" validate:"required"`
	// 协议参数
	Properties map[string]string `json:"properties" validate:"-"`

	//设备层驱动的引用
	DriverKey string `json:"driverKey"`

	//设备对应的协议
	PluginName string `json:"-" validate:"-"`
}

// Validate 核心配置文件校验
func (c Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// 是否处于虚拟运行模式：未建立真实的设备连接
func IsVirtual() bool {
	return os.Getenv(ENV_VIRTUAL) == "true"
}

// 网关元数据
type Metadata struct {
	// SerialNo 网关在云端定义的SN号，用于唯一标识网关设备
	SerialNo string `json:"serialNo"`

	//产品型号，标识网关的硬件型号和规格
	//示例值:
	// - "GW1000": 基础型号，支持基础协议和功能
	// - "GW2000": 高级型号，支持所有协议和扩展功能
	//注意:
	// 1. 型号决定了网关支持的功能和性能参数
	// 2. 不同型号可能有不同的固件版本要求
	Model string `json:"model"`

	//厂商名称，标识网关的生产厂商
	//示例值: "Huawei", "ZTE", "Cisco"
	//注意:
	// 1. 厂商名称应与官方注册名称一致
	// 2. 用于区分不同厂商的设备兼容性和支持服务
	Vendor string `json:"vendor"`

	// 网关硬件唯一编号
	HardwareSN string `json:"hardwareSn"`

	// 集成电路卡识别码，即 SIM 卡卡号
	ICCID string `json:"iccid"`

	//软件版本号，例如："1.0.0"
	SoftwareVersion string `json:"softwareVersion"`
}
