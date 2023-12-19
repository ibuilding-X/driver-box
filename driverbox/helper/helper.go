// 核心工具助手文件

package helper

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/helper/shadow"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"io/fs"
	"path/filepath"
	"sync"
)

var Exports []export.Export

var DeviceShadow shadow.DeviceShadow // 本地设备影子

var PluginCacheMap = &sync.Map{} // 插件通用缓存

var Crontab crontab.Crontab // 全局定时任务实例

var EnvConfig config.EnvConfig

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

func PointCacheFilter(deviceData *plugin.DeviceData) {
	// 定义一个空的整型数组
	var points []plugin.PointData
	for _, point := range deviceData.Values {
		// 获取点位信息
		_, ok := CoreCache.GetPointByDevice(deviceData.DeviceName, point.PointName)
		if !ok {
			Logger.Warn("unknown point", zap.Any("deviceName", deviceData.DeviceName), zap.Any("pointName", point.PointName))
			continue
		}
		// 缓存比较
		shadowValue, _ := DeviceShadow.GetDevicePoint(deviceData.DeviceName, point.PointName)
		if shadowValue == point.Value {
			Logger.Debug("point value = cache, stop sending to messageBus")
		} else {
			// 点位值类型名称转换
			points = append(points, point)
		}
		// 缓存
		if err := DeviceShadow.SetDevicePoint(deviceData.DeviceName, point.PointName, point.Value); err != nil {
			Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceName", deviceData.DeviceName))
		}
	}
	deviceData.Values = points
}
