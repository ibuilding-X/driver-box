package mirror

type ModelTemplate struct {
	ModeId   string             // 设备模型ID
	Capacity int                // 设备数量上限
	Points   []HoldingRegisters // 设备点位
}

// 浮点数占用2个寄存器，整数占用1个寄存器
type HoldingRegisters struct {
	Name string // 点位名称
}
