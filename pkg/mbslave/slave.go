package mbslave

import "sync"

const (
	MaxAddress        uint16 = 65535
	MaxLengthRegister uint16 = MaxAddress + 1
)

// slaveDevice 从站设备
type slaveDevice struct {
	mu               *sync.Mutex
	coils            interface{}               // 线圈（待实现）
	discreteInputs   interface{}               // 离散输入（待实现）
	inputRegisters   interface{}               // 输入寄存器（待实现）
	holdingRegisters [MaxLengthRegister]uint16 // 保持寄存器
}

// ReadHoldingRegisters 读取保持寄存器
func (s *slaveDevice) ReadHoldingRegisters(address, quantity uint16) (results []uint16, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 校验地址
	if address > MaxAddress {
		return nil, ErrIllegalDataAddress
	}

	// 校验数量
	if address+quantity > MaxLengthRegister {
		return nil, ErrIllegalDataAddress
	}

	return s.holdingRegisters[address : address+quantity], nil
}

// WriteSingleRegister 写单个保持寄存器
func (s *slaveDevice) WriteSingleRegister(address, value uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 校验地址
	if address > MaxAddress {
		return ErrIllegalDataAddress
	}

	// 写入
	s.holdingRegisters[address] = value
	return nil
}

// WriteMultipleRegisters 写多个保持寄存器
func (s *slaveDevice) WriteMultipleRegisters(address, quantity uint16, value []uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 校验地址
	if address > MaxAddress {
		return ErrIllegalDataAddress
	}

	// 校验数量
	if address+quantity > MaxLengthRegister {
		return ErrIllegalDataAddress
	}

	// 校验值数量
	if int(quantity) != len(value) {
		return ErrIllegalDataValue
	}

	// 循环写入
	for i := range quantity {
		s.holdingRegisters[address+i] = value[i]
	}
	return nil
}

func newSlaveDevice() *slaveDevice {
	return &slaveDevice{
		mu:               &sync.Mutex{},
		coils:            nil,
		discreteInputs:   nil,
		inputRegisters:   nil,
		holdingRegisters: [MaxLengthRegister]uint16{},
	}
}
