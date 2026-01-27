package library

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/fileutil"
)

type DeviceModel struct {
}

// 加载指定key的驱动
func (device *DeviceModel) LoadLibrary(modelKey string) (config.Model, error) {
	filePath := path.Join(config.ResourcePath, baseDir, string(deviceModel), modelKey+".json")
	if !fileutil.FileExists(filePath) {
		return config.Model{}, fmt.Errorf("device model library not found: %s", modelKey)
	}
	//读取filePath中的文件内容
	bytes, e := fileutil.ReadFileBytes(filePath)
	if e != nil {
		return config.Model{}, e
	}
	model := config.Model{}
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
