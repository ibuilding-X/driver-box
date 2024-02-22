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

func PointCacheFilter(deviceData *plugin.DeviceData) {
	// 定义一个空的整型数组
	var points []plugin.PointData
	for _, point := range deviceData.Values {
		// 获取点位信息
		p, ok := CoreCache.GetPointByDevice(deviceData.SN, point.PointName)
		if !ok {
			Logger.Warn("unknown point", zap.Any("deviceSn", deviceData.SN), zap.Any("pointName", point.PointName))
			continue
		}

		//数据类型纠正
		realValue, err := ConvPointType(point.Value, p.ValueType)
		if err != nil {
			Logger.Error("convert point value error", zap.Error(err), zap.Any("deviceSn", deviceData.SN),
				zap.String("pointName", p.Name), zap.Any("value", point.Value))
		} else {
			point.Value = realValue
		}

		// 缓存比较
		shadowValue, _ := DeviceShadow.GetDevicePoint(deviceData.SN, point.PointName)

		if p.ReportMode == config.ReportMode_Period {
			Logger.Debug("point report mode is period, stop to trigger ExportTo", zap.String("pointName", p.Name))
		} else if p.ReportMode == config.ReportMode_Change && shadowValue == point.Value {
			Logger.Debug("point report mode is change, stop to trigger ExportTo", zap.String("pointName", p.Name))
		} else {
			// 点位值类型名称转换
			points = append(points, point)
		}

		// 缓存
		if err := DeviceShadow.SetDevicePoint(deviceData.SN, point.PointName, point.Value); err != nil {
			Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceSn", deviceData.SN))
		}
	}
	deviceData.Values = points
	deviceData.ExportType = plugin.RealTimeExport
}

// 触发事件
func TriggerEvents(eventCode string, key string, value interface{}) {
	for _, export0 := range Exports {
		if !export0.IsReady() {
			Logger.Warn("export not ready")
			continue
		}
		err := export0.OnEvent(eventCode, key, value)
		if err != nil {
			Logger.Error("trigger event error", zap.String("eventCode", eventCode), zap.String("key", key), zap.Any("value", value), zap.Error(err))
		}
	}
}
