package mbslave

import "sync"

type Handler interface {
	// ReadHoldingRegisters 读取保持寄存器
	ReadHoldingRegisters(unitID uint8, address, quantity uint16) (results []uint16, err error)
	// WriteSingleRegister 写单个保持寄存器
	WriteSingleRegister(unitID uint8, address, value uint16) error
	// WriteMultipleRegisters 写多个保持寄存器
	WriteMultipleRegisters(unitID uint8, address, quantity uint16, value []uint16) error
}

type handlerImpl struct {
	slaves *sync.Map // 从站设备
}

func (h *handlerImpl) ReadHoldingRegisters(unitID uint8, address, quantity uint16) (results []uint16, err error) {
	slave, err := h.GetSlaveDevice(unitID)
	if err != nil {
		return nil, err
	}

	return slave.ReadHoldingRegisters(address, quantity)
}

func (h *handlerImpl) WriteSingleRegister(unitID uint8, address, value uint16) error {
	slave, err := h.GetSlaveDevice(unitID)
	if err != nil {
		return err
	}

	return slave.WriteSingleRegister(address, value)
}

func (h *handlerImpl) WriteMultipleRegisters(unitID uint8, address, quantity uint16, value []uint16) error {
	slave, err := h.GetSlaveDevice(unitID)
	if err != nil {
		return err
	}

	return slave.WriteMultipleRegisters(address, quantity, value)
}

// InitSlaveDevice 初始化从站设备
func (h *handlerImpl) InitSlaveDevice(unitID uint8) {
	if _, ok := h.slaves.Load(unitID); ok {
		return
	}

	h.slaves.Store(unitID, newSlaveDevice())
}

func (h *handlerImpl) GetSlaveDevice(unitID uint8) (slave *slaveDevice, err error) {
	slaveAny, ok := h.slaves.Load(unitID)
	if !ok {
		return nil, ErrBadUnitId
	}

	slave, ok = slaveAny.(*slaveDevice)
	if !ok {
		return nil, ErrSlaveDeviceNotFound
	}
	return
}

func newHandlerImpl() *handlerImpl {
	return &handlerImpl{
		slaves: &sync.Map{},
	}
}

func NewHandler() Handler {
	return newHandlerImpl()
}
