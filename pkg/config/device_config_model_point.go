package config

import "encoding/json"

type PointEnum struct {
	//枚举名称
	Name string `json:"name"`
	//枚举值
	Value interface{} `json:"value"`
	//枚举图标：用于界面展示
	Icon string `json:"icon"`
}

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
