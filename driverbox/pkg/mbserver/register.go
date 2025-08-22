package mbserver

//
//import (
//	"errors"
//)
//
//type Register interface {
//	Initialize(models []Model, devices []Device) error                                        // 初始化
//	SetProperty(did string, property string, value interface{}) error                         // 设置属性
//	GetProperty(did string, property string) (interface{}, error)                             // 获取属性
//	ParseAddress(address uint16) (id string, property string, valueType ValueType, err error) // 解析寄存器地址
//	Read(address, quantity uint16) (results []uint16, err error)                              // 读寄存器
//	Write(address, value uint16) error                                                        // 写寄存器
//}
//
//type registerImpl struct {
//	node *registerNode
//}
//
//func (r *registerImpl) Initialize(models []Model, devices []Device) error {
//	node, err := newRegisterNode(models, devices)
//	if err != nil {
//		return err
//	}
//	r.node = node
//	return nil
//}
//
//func (r *registerImpl) SetProperty(did string, property string, value interface{}) error {
//	if r.node == nil {
//		return errors.New("register not initialized")
//	}
//	return r.node.SetProperty(did, property, value)
//}
//
//func (r *registerImpl) GetProperty(did string, property string) (interface{}, error) {
//	if r.node == nil {
//		return nil, errors.New("register not initialized")
//	}
//	return r.node.GetProperty(did, property)
//}
//
//func (r *registerImpl) ParseAddress(address uint16) (id string, property string, valueType ValueType, err error) {
//	if r.node == nil {
//		return "", "", 0, errors.New("register not initialized")
//	}
//	return r.node.ParseAddress(address)
//}
//
//func (r *registerImpl) Read(address, quantity uint16) (results []uint16, err error) {
//	if r.node == nil {
//		return nil, errors.New("register not initialized")
//	}
//	return r.node.Get(address, quantity)
//}
//
//func (r *registerImpl) Write(address, value uint16) error {
//	if r.node == nil {
//		return errors.New("register not initialized")
//	}
//	return r.node.Set(address, value)
//}
//
//func NewRegister() Register {
//	return &registerImpl{}
//}
