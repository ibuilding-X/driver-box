package modbus

import (
	"errors"
)

type Register interface {
	Initialize(models []Model, devices []Device) error                    // 初始化
	SetProperty(did string, property string, value interface{}) error     // 设置属性
	GetProperty(did string, property string) (interface{}, error)         // 获取属性
	ParseAddress(address uint16) (did string, property string, err error) // 解析寄存器地址
}

type registerImpl struct {
	node *registerNode
}

func (r *registerImpl) Initialize(models []Model, devices []Device) error {
	node, err := newRegisterNode(models, devices)
	if err != nil {
		return err
	}
	r.node = node
	return nil
}

func (r *registerImpl) SetProperty(did string, property string, value interface{}) error {
	if r.node == nil {
		return errors.New("register not initialized")
	}
	return r.node.SetProperty(did, property, value)
}

func (r *registerImpl) GetProperty(did string, property string) (interface{}, error) {
	if r.node == nil {
		return nil, errors.New("register not initialized")
	}
	return r.node.GetProperty(did, property)
}

func (r *registerImpl) ParseAddress(address uint16) (did string, property string, err error) {
	if r.node == nil {
		return "", "", errors.New("register not initialized")
	}
	return r.node.ParseAddress(address)
}

func NewRegister() Register {
	return &registerImpl{}
}
