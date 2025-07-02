package mbslave

import "github.com/go-playground/validator/v10"

var validate = validator.New()

type Model struct {
	ID             string               `validate:"required,max=64"` // 模型 ID
	Properties     []Property           `validate:"required"`        // 属性列表
	registerLength uint16               // 寄存器长度，表示存储该设备所有属性所需的寄存器数量
	property       map[string]*Property // 属性映射表
}

// Validate 验证 Model
func (m *Model) Validate() error {
	return validate.Struct(m)
}

// pretreatment 模型预处理
// 1. 计算所需寄存器长度
// 2. 属性映射
func (m *Model) pretreatment() *Model {
	m.registerLength = 0
	m.property = make(map[string]*Property)

	for _, property := range m.Properties {
		var length uint16 = 1

		switch property.ValueType {
		case "float32":
			length = 2
		}

		m.property[property.Name] = &Property{
			Name: property.Name,
			// Mode:      property.Mode,
			ValueType: property.ValueType,
			address:   m.registerLength,
			length:    length,
		}

		m.registerLength = m.registerLength + length
	}

	return m
}

type Property struct {
	Name string `validate:"required,max=100"` // 属性名称
	//Mode      string `validate:"required,oneof=r w rw"`         // 读写模式
	ValueType string `validate:"required,oneof=uint16 float32"` // 属性值类型
	address   uint16 // 地址，表示该属性值位于属性数组中的起始地址
	length    uint16 // 长度，表示该属性值占用的寄存器数量
}

type PropertyValue struct {
	Mid      string      `validate:"required,max=64"`  // 模型 ID
	Did      string      `validate:"required,max=100"` // 设备 ID
	Property string      `validate:"required,max=100"` // 属性名称
	Value    interface{} `validate:""`                 // 属性值
}
