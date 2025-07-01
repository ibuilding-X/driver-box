package mbslave

import "sync"

const (
	MaxAddress        uint16 = 65535
	MaxLengthRegister uint16 = MaxAddress + 1
)

type Handler interface {
	// ReadHoldingRegisters 读取保持寄存器
	ReadHoldingRegisters(unitID uint8, address, quantity uint16) (results []uint16, err error)
	// WriteSingleRegister 写单个保持寄存器
	WriteSingleRegister(unitID uint8, address, value uint16) error
	// WriteMultipleRegisters 写多个保持寄存器
	WriteMultipleRegisters(unitID uint8, address, quantity uint16, value []uint16) error
}

// slave 从站数据结构
type slave struct {
	coils            interface{}               // 待实现
	discreteInputs   interface{}               // 待实现
	inputRegisters   interface{}               // 待实现
	holdingRegisters [MaxLengthRegister]uint16 // 保持寄存器
}

type handlerImpl struct {
	slaves map[uint8]*slave
	mu     *sync.Mutex
}

func (h *handlerImpl) ReadHoldingRegisters(unitID uint8, address, quantity uint16) (results []uint16, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, ok := h.slaves[unitID]
	if !ok {
		return nil, ErrBadUnitId
	}

	if address > MaxAddress {
		return nil, ErrIllegalDataAddress
	}

	if address+quantity > MaxLengthRegister {
		return nil, ErrIllegalDataAddress
	}

	return data.holdingRegisters[address : address+quantity], nil
}

func (h *handlerImpl) WriteSingleRegister(unitID uint8, address, value uint16) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, ok := h.slaves[unitID]
	if !ok {
		return ErrBadUnitId
	}

	if address > MaxAddress {
		return ErrIllegalDataAddress
	}

	data.holdingRegisters[address] = value
	return nil
}

func (h *handlerImpl) WriteMultipleRegisters(unitID uint8, address, quantity uint16, value []uint16) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, ok := h.slaves[unitID]
	if !ok {
		return ErrBadUnitId
	}

	if address > MaxAddress {
		return ErrIllegalDataAddress
	}

	if address+quantity > MaxLengthRegister {
		return ErrIllegalDataAddress
	}

	if int(quantity) != len(value) {
		return ErrIllegalDataValue
	}

	for i := uint16(0); i < quantity; i++ {
		data.holdingRegisters[address+i] = value[i]
	}
	return nil
}

func (h *handlerImpl) InitSlave(unitID uint8) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.slaves[unitID]; ok {
		return
	}

	h.slaves[unitID] = newSlave()
}

func newSlave() *slave {
	return &slave{
		coils:            nil,
		discreteInputs:   nil,
		inputRegisters:   nil,
		holdingRegisters: [MaxLengthRegister]uint16{},
	}
}

func newHandlerImpl() *handlerImpl {
	return &handlerImpl{
		slaves: make(map[uint8]*slave),
		mu:     &sync.Mutex{},
	}
}

func NewHandler() Handler {
	return newHandlerImpl()
}
