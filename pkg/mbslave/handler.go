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
	slave := h.InitSlaveDevice(unitID)
	return slave.ReadHoldingRegisters(address, quantity)
}

func (h *handlerImpl) WriteSingleRegister(unitID uint8, address, value uint16) error {
	slave := h.InitSlaveDevice(unitID)
	return slave.WriteSingleRegister(address, value)
}

func (h *handlerImpl) WriteMultipleRegisters(unitID uint8, address, quantity uint16, value []uint16) error {
	slave := h.InitSlaveDevice(unitID)
	return slave.WriteMultipleRegisters(address, quantity, value)
}

// InitSlaveDevice 初始化从站设备
func (h *handlerImpl) InitSlaveDevice(unitID uint8) *slaveDevice {
	if slaveAny, ok := h.slaves.Load(unitID); ok {
		return slaveAny.(*slaveDevice)
	}

	slave := newSlaveDevice(unitID)
	h.slaves.Store(unitID, slave)
	return slave
}

// GetSlaveDevice 获取从站设备
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
