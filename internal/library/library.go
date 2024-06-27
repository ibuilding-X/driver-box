package library

import (
	glua "github.com/yuin/gopher-lua"
	"sync"
)

type Type string

const (
	//设备层驱动
	deviceDriver Type = "driver"
	//物模型
	DeviceModel Type = "model"
	//协议层驱动
	ProtocolDriver Type = "protocol"

	//镜像设备模版
	mirrorTemplate Type = "mirror_tpl"
)

var driverOnce = &sync.Once{}
var mirrorOnce = &sync.Once{}
var driver *DeviceDriver
var mirror *MirrorTemplate

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
