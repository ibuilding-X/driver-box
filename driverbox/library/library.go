package library

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	glua "github.com/yuin/gopher-lua"
	"path"
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

	ProtocolConfigKey = "protocolKey"
	DriverConfigKey   = "driverKey"
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
			drivers: &sync.Map{},
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

// 加载library中的内容
func LoadContent(library string, key string) ([]byte, error) {
	filePath := path.Join(config.ResourcePath, baseDir, library, key+".json")
	if !common.FileExists(filePath) {
		return []byte{}, fmt.Errorf("library not found: %s/%s.json", library, key)
	}
	//读取filePath中的文件内容
	return common.ReadFileBytes(filePath)
}

// 加载library中的内容，并填充至结构体
func LoadLibrary(library string, key string, v any) error {
	bytes, err := LoadContent(library, key)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, v)
}
