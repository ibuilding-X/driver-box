package library

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	glua "github.com/yuin/gopher-lua"
	"path"
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

var baseDir = path.Join(config.ResourcePath, "library")

var once = &sync.Once{}
var device *DeviceDriver
var mirror *MirrorTemplate

// 设备驱动库
func Driver() *DeviceDriver {
	once.Do(func() {
		device = &DeviceDriver{
			drivers: make(map[string]*glua.LState),
		}
	})
	return device
}

// 镜像模版库
func Mirror() *MirrorTemplate {
	once.Do(func() {
		mirror = &MirrorTemplate{}
	})
	return mirror
}
