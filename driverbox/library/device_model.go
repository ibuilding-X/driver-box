package library

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
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

// 列出所有物模型
func (device *DeviceModel) ListModels() []string {
	modelPath := path.Join(config.ResourcePath, baseDir, string(deviceModel))
	//获取 modelPath目录下的所有json文件名
	var files []string
	_ = filepath.WalkDir(modelPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			files = append(files, strings.TrimRight(d.Name(), ".json"))
		}
		return nil
	})
	return files

}
