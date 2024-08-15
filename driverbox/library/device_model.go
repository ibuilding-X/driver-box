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
func (device *DeviceModel) LoadLibrary(modelKey string) (config.DeviceModel, error) {
	filePath := path.Join(config.ResourcePath, baseDir, string(deviceModel), modelKey+".json")
	if !common.FileExists(filePath) {
		return config.DeviceModel{}, fmt.Errorf("device model library not found: %s", modelKey)
	}
	//读取filePath中的文件内容
	bytes, e := common.ReadFileBytes(filePath)
	if e != nil {
		return config.DeviceModel{}, e
	}
	model := config.DeviceModel{}
	e = json.Unmarshal(bytes, &model)
	return model, e
}
