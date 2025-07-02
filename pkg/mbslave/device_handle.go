package mbslave

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type DeviceHandler interface {
	Handler
	ImportModels(models []Model) error                   // 导入设备模型
	SetProperty(unitID uint8, value PropertyValue) error // 设置设备属性
}

type device struct {
	values []uint16 // 属性值列表
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
		// 模型校验
		if err := model.Validate(); err != nil {
			return err
		}
		if _, ok := d.models.Load(model.ID); ok {
			continue
		}
		// 存储（存储与处理后的模型）
		d.models.Store(model.ID, model.pretreatment())
	}
	return nil
}

func (d *deviceHandlerImpl) SetProperty(unitID uint8, value PropertyValue) error {
	// 初始化从站设备
	slave := d.handler.InitSlaveDevice(unitID)

	// 查询模型
	model, err := d.GetModel(value.Mid)
	if err != nil {
		return err
	}

	// 查询属性信息
	property, ok := model.property[value.Property]
	if !ok {
		return ErrPropertyNotFound
	}

	// 查询设备
	dev, err := d.GetDevice(value.Did)
	if err != nil { // 设备不存在，创建设备
		// 计算设备占用寄存器长度、起始位、结束位
		start := d.needle.Load()
		length := model.registerLength
		end := start + uint32(length)

		if !d.needle.CompareAndSwap(start, end) {
			return ErrSetProperty
		}

		dev = &device{
			values: slave.holdingRegisters[start:end],
		}
		d.devices.Store(value.Did, dev)
	}

	// 属性值转换
	u16Slice, err := convUint16s(property.ValueType, value.Value)
	if err != nil {
		return err
	}

	// 转化后长度校验
	if len(u16Slice) != int(property.length) {
		return fmt.Errorf("length error after property conversion")
	}

	// 获取属性值存储区域
	propertyValue := dev.values[property.address : property.address+property.length]

	// 写入值
	slave.mu.Lock()
	defer slave.mu.Unlock()
	for i, u := range u16Slice {
		propertyValue[i] = u
	}

	return nil
}

// GetModel 获取模型
func (d *deviceHandlerImpl) GetModel(mid string) (*Model, error) {
	modelAny, ok := d.models.Load(mid)
	if !ok {
		return nil, ErrModelNotFound
	}

	model, ok := modelAny.(*Model)
	if !ok {
		return nil, ErrModelNotFound
	}
	return model, nil
}

// GetDevice 获取设备
func (d *deviceHandlerImpl) GetDevice(did string) (*device, error) {
	deviceAny, ok := d.devices.Load(did)
	if !ok {
		return nil, ErrDeviceNotFound
	}

	dev, ok := deviceAny.(*device)
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return dev, nil
}

func NewDeviceHandler() DeviceHandler {
	return &deviceHandlerImpl{
		handler: newHandlerImpl(),
		models:  &sync.Map{},
		devices: &sync.Map{},
		needle:  &atomic.Uint32{},
	}
}
