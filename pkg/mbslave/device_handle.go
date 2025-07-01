package mbslave

import (
	"sync"
	"sync/atomic"
)

type DeviceHandler interface {
	Handler
	ImportModels(models []Model) error                   // 导入设备模型
	SetProperty(unitID uint8, value PropertyValue) error // 设置设备属性
}

type device struct {
	values []uint16
}

type deviceHandlerImpl struct {
	handler *handlerImpl
	models  *sync.Map      // 设备模型
	devices *sync.Map      // 设备
	needle  *atomic.Uint32 // 当前寄存器索引（默认为 0）
}

func (d *deviceHandlerImpl) ReadHoldingRegisters(unitID uint8, address, quantity uint16) (results []uint16, err error) {
	return d.handler.ReadHoldingRegisters(unitID, address, quantity)
}

func (d *deviceHandlerImpl) WriteSingleRegister(unitID uint8, address, value uint16) error {
	return d.handler.WriteSingleRegister(unitID, address, value)
}

func (d *deviceHandlerImpl) WriteMultipleRegisters(unitID uint8, address, quantity uint16, value []uint16) error {
	return d.handler.WriteMultipleRegisters(unitID, address, quantity, value)
}

func (d *deviceHandlerImpl) ImportModels(models []Model) error {
	for _, model := range models {
		complementModel(&model)
		if _, ok := d.models.Load(model.ID); ok {
			continue
		}
		d.models.Store(model.ID, &model)
	}
	return nil
}

func (d *deviceHandlerImpl) SetProperty(unitID uint8, value PropertyValue) error {
	// 查询模型
	modelAny, ok := d.models.Load(value.Mid)
	if !ok {
		return ErrModelNotFound
	}
	_, _ = modelAny.(*Model)

	// 查询 slave
	d.handler.InitSlaveDevice(unitID)

	// todo something

	return nil
}

func NewDeviceHandler() DeviceHandler {
	return &deviceHandlerImpl{
		handler: newHandlerImpl(),
		models:  &sync.Map{},
		devices: &sync.Map{},
		needle:  &atomic.Uint32{},
	}
}
