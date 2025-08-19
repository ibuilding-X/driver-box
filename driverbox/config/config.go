// Package config 核心配置
package config

import (
	"encoding/json"
	"fmt"
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

type PointMap map[string]interface{} // 点位 Map，可转换为标准点位数据

// Name 获取点位名称
func (pm PointMap) Name() string {
	return pm["name"].(string)
}

// ReadWrite 获取点位读写模式
func (pm PointMap) ReadWrite() ReadWrite {
	return ReadWrite(fmt.Sprintf("%s", pm["readWrite"]))
}

func (pm PointMap) FieldValue(key string) interface{} {
	return pm[key]
}

// ToPoint 转换为标准点位数据
func (pm PointMap) ToPoint() Point {
	p := Point{
		Extends: make(map[string]interface{}),
	}

	for key, v := range pm {
		switch key {
		case "name":
			p.Name = fmt.Sprintf("%s", v)
		case "description":
			p.Description = fmt.Sprintf("%s", v)
		case "valueType":
			p.ValueType = ValueType(fmt.Sprintf("%s", v))
		case "readWrite":
			p.ReadWrite = ReadWrite(fmt.Sprintf("%s", v))
		case "units":
			p.Units = fmt.Sprintf("%s", v)
		case "reportMode":
			p.ReportMode = ReportMode(fmt.Sprintf("%s", v))
		case "scale":
			p.Scale = v.(float64)
		case "decimals":
			switch v.(type) {
			case float64:
				p.Decimals = int(v.(float64))
			default:
				p.Decimals = v.(int)
			}
		case "enums":
			enums := make([]PointEnum, 0)
			b, err := json.Marshal(v)
			if err == nil {
				json.Unmarshal(b, &enums)
				p.Enums = enums
			}

			//p.Enums = v.([]PointEnum)
		default:
			p.Extends[key] = v
		}
	}
	//浮点数，且未指定decimals，默认未2
	if p.ValueType == ValueType_Float && pm["decimals"] == nil {
		p.Decimals = 2
	}
	return p
}

// Config 配置
type Config struct {
	// 设备模型
	DeviceModels []DeviceModel `json:"deviceModels" validate:""`
	// 连接配置
	Connections map[string]interface{} `json:"connections" validate:""`
	// 协议名称（通过协议名称区分连接模式：客户端、服务端）
	ProtocolName string `json:"protocolName" validate:"required"`
	// 配置唯一key，一般对应目录名称
	Key string `json:"-" validate:"-"`
	// 模型索引
	modelIndexes map[string]int
}

//------------------------------ 设备模型 ------------------------------

type Model struct {
	ModelBase
	Points  map[string]Point  `json:"points"`
	Devices map[string]Device `json:"devices"`
}

// ModelBase 模型基础信息
type ModelBase struct {
	// 模型名称
	Name string `json:"name" validate:"required"`
	// 云端模型 ID
	ModelID string `json:"modelId" validate:"required"`
	// 模型描述
	Description string `json:"description" validate:"required"`
	//扩展属性
	Attributes map[string]interface{} `json:"attributes"`
}

// DeviceModel 设备模型
type DeviceModel struct {
	ModelBase
	// 模型点位列表
	DevicePoints []PointMap `json:"devicePoints" validate:"required"`
	// 设备列表
	Devices []Device `json:"devices" validate:"required"`
	// 设备索引
	deviceIndexes map[string]int
}

// Point 点位数据
type Point struct {
	// 点位名称
	Name string `json:"name" validate:"required"`
	// 点位描述
	Description string `json:"description" validate:"required"`
	// 值类型
	ValueType ValueType `json:"valueType" validate:"required,oneof=int float string"`
	// 读写模式
	ReadWrite ReadWrite `json:"readWrite" validate:"required,oneof=R W RW"`
	// 单位
	Units string `json:"units" validate:"-"`
	// 上报模式
	ReportMode ReportMode `json:"reportMode" validate:"required"`
	//数值精度
	Scale float64 `json:"scale"`

	//保留小数位数
	Decimals int `json:"decimals"`

	//点位枚举表
	Enums []PointEnum `json:"enums"`

	// 扩展参数
	Extends map[string]interface{} `json:"-" validate:"-"`
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
}

// DeviceBusinessProp 设备业务属性
type DeviceBusinessProp struct {
	// SN 设备 SN
	SN string
	// ParentID 父设备 ID
	ParentID string
	// SystemID 所属系统 ID
	SystemID string
}

// TimerTask 定时任务
type TimerTask struct {
	// 间隔（单位：毫秒）
	Interval string `json:"interval" validate:"required"`
	// 任务类型
	Type string `json:"type" validate:"required"`
	// 任务动作
	Action interface{} `json:"action" validate:"required"`
}

type ReadPointsAction struct {
	//设备SN列表
	Devices []string `json:"devices"`
	Points  []string `json:"points"`
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

// UpdateIndexAndClean 更新索引并清理无效数据
// 1. 更新模型、设备索引
// 2. 移除无效连接
func (c Config) UpdateIndexAndClean() Config {
	c.modelIndexes = make(map[string]int)

	usefulConnKeys := make(map[string]struct{})
	// 遍历模型
	for i, _ := range c.DeviceModels {
		c.modelIndexes[c.DeviceModels[i].Name] = i
		c.DeviceModels[i].deviceIndexes = make(map[string]int)
		// 遍历设备
		for j, _ := range c.DeviceModels[i].Devices {
			usefulConnKeys[c.DeviceModels[i].Devices[j].ConnectionKey] = struct{}{}
			c.DeviceModels[i].deviceIndexes[c.DeviceModels[i].Devices[j].ID] = j
		}
	}

	// 移除无效连接
	for k, _ := range c.Connections {
		// 是否为自动发现连接
		if connIsSupportDiscover(c.Connections[k]) {
			continue
		}
		// 是否为有效连接
		if _, ok := usefulConnKeys[k]; ok {
			continue
		}
		// 移除无效连接
		delete(c.Connections, k)
	}

	return c
}

// GetModelIndexes 获取模型索引
func (c Config) GetModelIndexes() map[string]int {
	return c.modelIndexes
}

// GetDeviceIndexes 获取设备索引
func (dm DeviceModel) GetDeviceIndexes() map[string]int {
	return dm.deviceIndexes
}

// connIsSupportDiscover 连接是否支持自动发现设备
func connIsSupportDiscover(conn any) bool {
	if m, ok := conn.(map[string]interface{}); ok {
		if _, ok := m["discover"]; ok && (m["discover"] == true || m["discover"] == "true" || m["discover"] == 1 || m["discover"] == "1") {
			return true
		}
	}
	return false
}

func (dm DeviceModel) ToModel() Model {
	points := make(map[string]Point)
	for _, pointMap := range dm.DevicePoints {
		point := pointMap.ToPoint()
		if point.Name != "" {
			points[point.Name] = point
		}
	}

	devices := make(map[string]Device)
	for i, _ := range dm.Devices {
		devices[dm.Devices[i].ID] = dm.Devices[i]
	}

	return Model{
		ModelBase: dm.ModelBase,
		Points:    points,
		Devices:   devices,
	}
}

// 网关元数据
type Metadata struct {
	//网关序列号，用于唯一标识网关设备，通常由厂商提供
	//示例值: "GW2023123456789"
	//注意:
	// 1. 序列号应与设备实物标签一致
	// 2. 序列号格式通常为: 厂商代码(2字母) + 年份(4数字) + 序列号(6数字)
	// 3. 序列号在系统内必须唯一，不可重复
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
}
