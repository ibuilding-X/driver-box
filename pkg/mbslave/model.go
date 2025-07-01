package mbslave

type Model struct {
	ID             string               `validate:"required,max=64"` // 模型 ID
	Properties     []Property           `validate:"required"`        // 属性列表
	registerNumber uint16               // 寄存器数量
	property       map[string]*Property // 映射表
}

type Property struct {
	Name      string `validate:"required,max=100"`              // 属性名称
	Mode      string `validate:"required,oneof=r w rw"`         // 读写模式
	ValueType string `validate:"required,oneof=uint16 float32"` // 属性值类型
	startAddr uint16 // 起始地址（默认从 0 开始）
	length    uint16 // 占用寄存器长度
}

type PropertyValue struct {
	Mid      string      `validate:"required,max=64"`  // 模型 ID
	Did      string      `validate:"required,max=100"` // 设备 ID
	Property string      `validate:"required,max=100"` // 属性名称
	Value    interface{} `validate:""`                 // 属性值
}
