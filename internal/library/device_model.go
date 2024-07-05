package library

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"path"
)

type DeviceModel struct {
}

// 加载指定key的驱动
func (device *DeviceModel) LoadLibrary(driverKey string) (config.Model, error) {
	filePath := path.Join(config.ResourcePath, baseDir, string(deviceModel), driverKey+".json")
	if !common.FileExists(filePath) {
		return config.Model{}, fmt.Errorf("mirror template not found: %s", driverKey)
	}
	//读取filePath中的文件内容
	bytes, e := common.ReadFileBytes(filePath)
	if e != nil {
		return config.Model{}, e
	}
	model := config.Model{}
	e = json.Unmarshal(bytes, &model)
	return model, e
}
