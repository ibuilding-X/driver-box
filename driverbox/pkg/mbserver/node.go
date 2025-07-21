package mbserver

import (
	"fmt"
	"math"
	"sync"
)

// registerNode 寄存器节点
// todo：优化存储字段，减少内存占用
type registerNode struct {
	index     uint16        // 索引
	address   uint16        // 寄存器起始地址
	did       *string       // 设备 ID
	property  *string       // 属性名称
	valueType ValueType     // 值类型
	access    Access        // 访问权限
	value     []uint16      // 值
	left      *registerNode // 左子节点
	right     *registerNode // 右子节点
	mu        sync.Mutex
}

// Insert 插入节点
func (r *registerNode) Insert(index uint16, address uint16, did *string, property *string, valueType ValueType, access Access) {
	switch {
	case index == r.index:
		r.address = address
		r.did = did
		r.property = property
		r.valueType = valueType
		r.access = access
		r.value = make([]uint16, calcValueTypeLength(valueType))
	case index < r.index:
		if r.left == nil {
			r.left = createRegisterNode(index, address, did, property, valueType, access)
		} else {
			r.left.Insert(index, address, did, property, valueType, access)
		}
	case index > r.index:
		if r.right == nil {
			r.right = createRegisterNode(index, address, did, property, valueType, access)
		} else {
			r.right.Insert(index, address, did, property, valueType, access)
		}
	}
}

// SetProperty 设置属性值
func (r *registerNode) SetProperty(did string, property string, value interface{}) error {
	if did == *r.did && property == *r.property {
		r.mu.Lock()
		defer r.mu.Unlock()
		values, err := convAnyToUint16s(r.valueType, value)
		if err != nil {
			return err
		}
		copy(r.value, values)
		return nil
	}

	if r.left != nil {
		if err := r.left.SetProperty(did, property, value); err == nil {
			return nil
		}
	}

	if r.right != nil {
		if err := r.right.SetProperty(did, property, value); err == nil {
			return nil
		}
	}

	return fmt.Errorf("device [%s] property [%s] not found", did, property)
}

// GetProperty 获取属性值
func (r *registerNode) GetProperty(did, property string) (interface{}, error) {
	if did == *r.did && property == *r.property {
		r.mu.Lock()
		defer r.mu.Unlock()
		return convUint16sToAny(r.valueType, r.value)
	}

	if r.left != nil {
		if v, err := r.left.GetProperty(did, property); err == nil {
			return v, nil
		}
	}

	if r.right != nil {
		if v, err := r.right.GetProperty(did, property); err == nil {
			return v, nil
		}
	}

	return nil, fmt.Errorf("device [%s] property [%s] not found", did, property)
}

// Search 搜索节点
func (r *registerNode) Search(address uint16) (*registerNode, error) {
	if address >= r.address && address < r.address+uint16(len(r.value)) {
		return r, nil
	}
	if address < r.address {
		if r.left == nil {
			return nil, fmt.Errorf("address [%d] not found", address)
		} else {
			return r.left.Search(address)
		}
	} else {
		if r.right == nil {
			return nil, fmt.Errorf("address [%d] not found", address)
		} else {
			return r.right.Search(address)
		}
	}
}

// Get 获取寄存器值
func (r *registerNode) Get(address, quantity uint16) (results []uint16, err error) {
	// 检索节点
	node, err := r.Search(address)
	if err != nil {
		return nil, err
	}

	// 遍历读取值
	var values []uint16
	for {
		values = append(values, node.value...)
		if len(values) >= int(quantity) {
			break
		}
		if node.right == nil {
			break
		}
		node = node.right
	}
	if len(values) < int(quantity) {
		return nil, fmt.Errorf("address [%d] quantity [%d] not found", address, quantity)
	}

	return values[:quantity], nil
}

// Set 设置寄存器值
func (r *registerNode) Set(address, value uint16) error {
	// 检索节点
	node, err := r.Search(address)
	if err != nil {
		return err
	}

	node.mu.Lock()
	defer node.mu.Unlock()
	node.value[node.address-address] = value
	return nil
}

// ParseAddress 解析地址
func (r *registerNode) ParseAddress(address uint16) (did string, property string, valueType ValueType, err error) {
	node, err := r.Search(address)
	if err != nil {
		return "", "", 0, err
	}

	return *node.did, *node.property, node.valueType, nil
}

// createRegisterNode 创建寄存器节点
func createRegisterNode(index uint16, address uint16, did *string, property *string, valueType ValueType, access Access) *registerNode {
	return &registerNode{
		index:     index,
		address:   address,
		did:       did,
		property:  property,
		valueType: valueType,
		access:    access,
		value:     make([]uint16, calcValueTypeLength(valueType)),
	}
}

// newRegisterNode 初始化寄存器节点
func newRegisterNode(models []Model, devices []Device) (*registerNode, error) {
	// 模型存储所需长度
	var modelMap = make(map[string]Model)
	for _, model := range models {
		for _, property := range model.Properties {
			model.need += int(calcValueTypeLength(property.ValueType))
		}
		modelMap[model.Id] = model
	}

	// 总长度
	var registers int
	var properties int
	for _, device := range devices {
		model, ok := modelMap[device.Mid]
		if !ok {
			return nil, fmt.Errorf("device [%s] model [%s] not found", device.Id, device.Mid)
		}
		properties += len(model.Properties)
		registers += model.need
	}
	if registers > math.MaxUint16+1 {
		return nil, fmt.Errorf("storage length exceeds 65536")
	}

	node := &registerNode{
		index: 0,
	}

	// 插入节点
	var index, address uint16
	for _, device := range devices {
		model, _ := modelMap[device.Mid]
		for _, property := range model.Properties {
			node.Insert(index, address, &device.Id, &property.Name, property.ValueType, property.Access)
			index++
			address += uint16(calcValueTypeLength(property.ValueType))
		}
	}

	return node, nil
}
