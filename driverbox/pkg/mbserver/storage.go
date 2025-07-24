package mbserver

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/mbserver/internal"
	"math"
	"sync"
)

type Storage struct {
	EnableSort    bool                       // 是否启用属性排序（r、rw、w）
	registers     [65536]uint16              // 寄存器数据
	device        map[string]internal.Device // 设备列表
	models        []internal.Model           // 模型列表
	properties    []internal.Property        // 属性列表
	registerUnits []internal.RegisterUnit    // 寄存器映射关系
	converter
	mu sync.RWMutex
}

func (s *Storage) Initialize(models []Model, devices []Device) error {
	// 处理模型
	indexes := s.handleModels(models)
	// 处理设备
	if err := s.handleDevices(devices, indexes); err != nil {
		return err
	}
	return nil
}

func (s *Storage) Write(address uint16, values []uint16) error {
	if err := s.verifyAddressQuantity(address, uint16(len(values))); err != nil {
		return err
	}

	s.mu.Lock()
	copy(s.registers[address:int(address)+len(values)], values)
	s.mu.Unlock()

	return nil
}

func (s *Storage) Read(address uint16, quantity uint16) ([]uint16, error) {
	if err := s.verifyAddressQuantity(address, quantity); err != nil {
		return nil, err
	}

	result := make([]uint16, quantity)
	s.mu.RLock()
	copy(result, s.registers[address:int(address)+int(quantity)])
	s.mu.RUnlock()

	return result, nil
}

func (s *Storage) SetProperty(id string, property string, v interface{}) error {
	// 计算寄存器地址
	start, _, valueType, err := s.getPropertyInfo(id, property)
	if err != nil {
		return err
	}

	// 数据转换
	uint16Slice, err := s.converter.Uint16Slice(v, valueType)
	if err != nil {
		return err
	}

	// 写入寄存器
	return s.Write(start, uint16Slice)
}

func (s *Storage) GetProperty(id string, property string) (interface{}, error) {
	start, quantity, valueType, err := s.getPropertyInfo(id, property)
	if err != nil {
		return nil, err
	}

	// 读取
	values, err := s.Read(start, quantity)
	if err != nil {
		return nil, err
	}

	// 转换
	return s.converter.ConvUint16Slice(values, valueType)
}

func (s *Storage) RegisterInfo(address uint16) (id string, property string, err error) {
	if int(address) > len(s.registerUnits)-1 {
		err = fmt.Errorf("register [%d] not found", address)
		return
	}

	register := s.registerUnits[address]
	id = register.Id
	property = s.properties[register.Property].Name

	return
}

func (s *Storage) DeviceMap() interface{} {
	result := make([]DeviceUnit, 0)
	var device DeviceUnit
	propertyMap := make(map[uint16]struct{})
	for _, unit := range s.registerUnits {
		if device.Id != unit.Id {
			if device.Id != "" {
				// 保存
				result = append(result, device)
			}

			// 获取模型信息
			modelIndex := s.device[unit.Id].Model
			model := s.models[modelIndex]

			// 重置
			device.Id = unit.Id
			device.ModelId = model.Id
			device.ModelName = model.Name
			device.Properties = make([]PropertyUnit, 0)
		}

		// 属性去重
		if _, ok := propertyMap[unit.Property]; ok {
			continue
		} else {
			propertyMap[unit.Property] = struct{}{}
		}

		// 处理属性
		prop := s.properties[unit.Property]
		start := s.device[unit.Id].Start + prop.RelativeStartAddress
		humanAddress := fmt.Sprintf("%d", start)
		if prop.Quantity > 1 {
			humanAddress = fmt.Sprintf("%d~%d", start, start+prop.Quantity-1)
		}
		device.Properties = append(device.Properties, PropertyUnit{
			Name:         prop.Name,
			Description:  prop.Description,
			Type:         ValueTypeText(prop.ValueType),
			Access:       AccessText(prop.Access),
			StartAddress: start,
			Quantity:     prop.Quantity,
			HumanAddress: humanAddress,
		})
	}

	result = append(result, device)
	return result
}

func (s *Storage) getPropertyInfo(id string, property string) (start, quantity uint16, valueType int, err error) {
	// 查找设备
	device, ok := s.device[id]
	if !ok {
		err = fmt.Errorf("device [%s] not found", id)
		return
	}

	// 查找模型
	model := s.models[device.Model]

	// 查找属性
	propIndex, ok := model.Property[property]
	if !ok {
		err = fmt.Errorf("device [%s] property [%s] not found", id, property)
		return
	}
	prop := s.properties[propIndex]

	// 计算寄存器地址范围
	start = device.Start + prop.RelativeStartAddress
	quantity = prop.Quantity
	valueType = prop.ValueType
	return
}

func (s *Storage) verifyAddressQuantity(address uint16, quantity uint16) error {
	if quantity < 1 || quantity > 2000 {
		return fmt.Errorf("modbus: quantity '%d' must be between '%d' and '%d'", quantity, 1, 2000)
	}

	if int(address+quantity) > math.MaxUint16+1 {
		return fmt.Errorf("modbus: quantity '%d' is out of range", quantity)
	}

	return nil
}

func (s *Storage) calsValueTypeLength(valueType int) uint16 {
	switch valueType {
	case ValueTypeBool, ValueTypeUint8, ValueTypeInt8, ValueTypeUint16, ValueTypeInt16:
		return 1
	case ValueTypeUint32, ValueTypeInt32, ValueTypeFloat32:
		return 2
	case ValueTypeUint64, ValueTypeInt64, ValueTypeFloat64:
		return 4
	default:
		return 1
	}
}

func (s *Storage) handleModels(models []Model) map[string]uint16 {
	result := make(map[string]uint16)
	for i, model := range models {
		// 处理属性
		start, end, quantity, mapping := s.handleProperties(model.Properties)
		s.models = append(s.models, internal.Model{
			Id:                 model.Id,
			Name:               model.Name,
			Quantity:           quantity,
			PropertyIndexStart: start,
			PropertyIndexEnd:   end,
			Property:           mapping,
		})
		result[model.Id] = uint16(i)
	}
	return result
}

func (s *Storage) handleProperties(properties []Property) (start, end, quantity uint16, mapping map[string]uint16) {
	// 排序
	if s.EnableSort {
		var r, rw, w, other []Property
		for _, property := range properties {
			switch property.Access {
			case AccessRead:
				r = append(r, property)
			case AccessReadWrite:
				rw = append(rw, property)
			case AccessWrite:
				w = append(w, property)
			default:
				other = append(other, property)
			}
		}

		properties = nil
		properties = append(properties, r...)
		properties = append(properties, rw...)
		properties = append(properties, w...)
		properties = append(properties, other...)
	}

	// 存储
	start = uint16(len(s.properties))
	end = start
	mapping = make(map[string]uint16)
	for i, property := range properties {
		mapping[property.Name] = start + uint16(i)
		length := s.calsValueTypeLength(property.ValueType)
		s.properties = append(s.properties, internal.Property{
			Description:          property.Description,
			RelativeStartAddress: quantity,
			Quantity:             length,
			Name:                 property.Name,
			ValueType:            property.ValueType,
			Access:               property.Access,
		})
		quantity += length
		end++
	}

	return
}

func (s *Storage) handleDevices(devices []Device, modelIndexes map[string]uint16) error {
	// 处理设备
	var start uint16
	for _, device := range devices {
		// 获取模型索引
		index, ok := modelIndexes[device.ModelId]
		if !ok {
			return fmt.Errorf("device [%s] model [%s] not found", device.Id, device.ModelId)
		}

		// 获取模型
		model := s.models[index]

		s.device[device.Id] = internal.Device{
			Start: start,
			End:   start + model.Quantity,
			Model: index,
		}
		start += model.Quantity

		// 获取属性
		left := model.PropertyIndexStart // 左区间
		right := model.PropertyIndexEnd  // 右区间
		properties := s.properties[left:right]

		// 处理寄存器单元映射关系
		for i, property := range properties {
			// 计算属性长度
			length := s.calsValueTypeLength(property.ValueType)
			for j := uint16(0); j < length; j++ {
				s.registerUnits = append(s.registerUnits, internal.RegisterUnit{
					Id:       device.Id,
					Property: left + uint16(i),
				})
			}
		}
	}

	return nil
}

func NewStorage() *Storage {
	return &Storage{
		EnableSort:    false,
		registers:     [65536]uint16{},
		device:        make(map[string]internal.Device),
		models:        make([]internal.Model, 0),
		properties:    make([]internal.Property, 0),
		registerUnits: make([]internal.RegisterUnit, 0),
	}
}
