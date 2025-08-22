package internal

type Device struct {
	Start uint16 // 寄存器起始地址
	End   uint16 // 寄存器结束地址
	Model uint16 // 模型索引
}

type Model struct {
	Id                 string
	Name               string            // 可选
	Quantity           uint16            // 寄存器数量
	PropertyIndexStart uint16            // 属性起始索引
	PropertyIndexEnd   uint16            // 属性结束索引
	Property           map[string]uint16 // 属性名称与索引映射
}

type Property struct {
	Description          string `json:"description"` // 可选
	RelativeStartAddress uint16 // 属性相对起始地址
	Quantity             uint16 // 寄存器数量
	Name                 string // 属性名称
	ValueType            int    // 属性值类型
	Access               int    // 属性访问权限
}

type RegisterUnit struct {
	Id       string // 设备 ID
	Property uint16 // 属性索引
}
