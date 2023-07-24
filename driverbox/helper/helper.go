// 核心工具助手文件

package helper

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/helper/shadow"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
)

var Exports []export.Export

var DeviceShadow shadow.DeviceShadow // 本地设备影子

var PluginCacheMap = &sync.Map{} // 插件通用缓存

var Crontab crontab.Crontab // 全局定时任务实例

var DriverConfig config.DriverConfig // 驱动配置

// Map2Struct map 转 struct，用于解析连接器配置
// m：map[string]interface
// v：&struct{}
func Map2Struct(m interface{}, v interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// PointValueType2EdgeX 点位值类型转换为 EdgeX 数据类型
// int => Int64、float => Float64、string => String
func PointValueType2EdgeX(valueType string) string {
	switch strings.ToLower(valueType) {
	case "int":
		return common.ValueTypeInt64
	case "float":
		return common.ValueTypeFloat64
	case "string":
		return common.ValueTypeString
	default:
		return valueType
	}
}

// GetChildDir 获取指定路径下所有子目录
func GetChildDir(path string) (list []string, err error) {
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			list = append(list, path)
		}
		return nil
	})
	return
}

// GetChildDirName 获取指定路径下所有子目录名称
func GetChildDirName(path string) (list []string, err error) {
	err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			list = append(list, d.Name())
		}
		return nil
	})
	return
}
