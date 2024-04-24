// 核心工具助手文件

package helper

import (
	"encoding/json"
	"fmt"
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
			continue
		}

		//精度换算
		if p.Scale != 0 {
			realValue, err = multiplyWithFloat64(realValue, p.Scale)
			if err != nil {
				Logger.Error("multiplyWithFloat64 error", zap.Error(err), zap.Any("deviceSn", deviceData.SN))
				continue
			}
		}

		//仅浮点类型作小数保留位数加工
		if p.ValueType == config.ValueType_Float {
			realValue = fmt.Sprintf("%.*f", p.Decimals, realValue)
		}

		point.Value = realValue

		// 缓存比较
		shadowValue, _ := DeviceShadow.GetDevicePoint(deviceData.SN, point.PointName)

		// 如果是周期上报模式，且缓存中有值，停止触发
		if p.ReportMode == config.ReportMode_Change && shadowValue == point.Value {
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

func multiplyWithFloat64(value interface{}, scale float64) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v * scale, nil
	case int16:
		return float64(v) * scale, nil
	case uint16:
		return float64(v) * scale, nil
	case uint32:
		return float64(v) * scale, nil
	case int32:
		return float64(v) * scale, nil
	case int64:
		return float64(v) * scale, nil
	case uint64:
		return float64(v) * scale, nil
	case float32:
		return float64(v) * scale, nil
	default:
		return 0, fmt.Errorf("cannot multiply %T with float64", value)
	}
}
