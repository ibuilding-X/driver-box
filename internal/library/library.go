package library

import (
	glua "github.com/yuin/gopher-lua"
	"sync"
)

type Type string

const (
	baseDir = "library"
	//设备层驱动
	deviceDriver Type = "driver"
	//物模型
	deviceModel Type = "model"
	//协议层驱动
	protocolDriver Type = "protocol"

	//镜像设备模版
	mirrorTemplate Type = "mirror_tpl"
)

var driverOnce = &sync.Once{}
var mirrorOnce = &sync.Once{}
var protocolOnce = &sync.Once{}
var modelOnce = &sync.Once{}
var driver *DeviceDriver
var mirror *MirrorTemplate
var protocol *ProtocolDriver
var model *DeviceModel

// 设备驱动库
func Driver() *DeviceDriver {
	driverOnce.Do(func() {
		driver = &DeviceDriver{
			drivers: make(map[string]*glua.LState),
		}
	})
	return driver
}

// 镜像模版库
func Mirror() *MirrorTemplate {
	mirrorOnce.Do(func() {
		mirror = &MirrorTemplate{}
	})
	return mirror
}

// 通信协议层驱动库
func Protocol() *ProtocolDriver {
	protocolOnce.Do(func() {
		protocol = &ProtocolDriver{
			drivers: make(map[string]*glua.LState),
		}
	})
	return protocol
}

// 设备模型库
func Model() *DeviceModel {
	modelOnce.Do(func() {
		model = &DeviceModel{}
	})
	return model
}
