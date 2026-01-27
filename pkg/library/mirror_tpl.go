package library

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/fileutil"
)

type MirrorTemplate struct {
}

// 加载指定key的驱动
func (device *MirrorTemplate) LoadLibrary(key string) (map[string]interface{}, error) {
	filePath := path.Join(config.ResourcePath, baseDir, string(mirrorTemplate), key+".json")
	if !fileutil.FileExists(filePath) {
		return nil, fmt.Errorf("mirror template not found: %s", key)
	}
	//读取filePath中的文件内容
	bytes, e := fileutil.ReadFileBytes(filePath)
	if e != nil {
		return nil, e
	}
	var result map[string]interface{}
	e = json.Unmarshal(bytes, &result)
	return result, e
}
