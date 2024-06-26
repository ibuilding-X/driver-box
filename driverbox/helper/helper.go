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
	"github.com/ibuilding-x/driver-box/internal/library"
	"go.uber.org/zap"
	"strconv"
	"strings"
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
	// 设备层驱动，对点位进行预处理
	err := pointValueProcess(deviceData)
	if err != nil {
		Logger.Error("device driver process error", zap.Error(err))
		return
	}

	// 定义一个空的整型数组
	var points []plugin.PointData
	for _, point := range deviceData.Values {
		// 获取点位信息
		p, ok := CoreCache.GetPointByDevice(deviceData.ID, point.PointName)
		if !ok {
			Logger.Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", point.PointName))
			continue
		}

		// 缓存比较
		shadowValue, _ := DeviceShadow.GetDevicePoint(deviceData.ID, point.PointName)

		// 如果是周期上报模式，且缓存中有值，停止触发
		if p.ReportMode == config.ReportMode_Change && shadowValue == point.Value {
			Logger.Debug("point report mode is change, stop to trigger ExportTo", zap.String("pointName", p.Name))
		} else {
			// 点位值类型名称转换
			points = append(points, point)
		}

		// 缓存
		if Logger != nil {
			Logger.Info("shadow store point value", zap.String("pointName", p.Name), zap.Any("value", point.Value))
		}
		if err := DeviceShadow.SetDevicePoint(deviceData.ID, point.PointName, point.Value); err != nil {
			Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceId", deviceData.ID))
		}
	}
	deviceData.Values = points
	deviceData.ExportType = plugin.RealTimeExport
}

func pointValueProcess(deviceData *plugin.DeviceData) error {
	device, ok := CoreCache.GetDevice(deviceData.ID)
	if !ok {
		Logger.Error("unknown device", zap.Any("deviceId", deviceData.ID))
		return fmt.Errorf("unknown device")
	}
	driverEnable := len(device.DriverKey) > 0

	//通过设备层驱动对点位值进行加工
	if driverEnable {
		result := library.DeviceDecode(device.DriverKey, library.DeviceDecodeRequest{DeviceId: deviceData.ID, Points: deviceData.Values})
		if result.Error != nil {
			Logger.Error("library.DeviceDecode error", zap.Error(result.Error), zap.Any("deviceData", deviceData))
			return result.Error
		} else {
			deviceData.Values = result.Points
		}
	}
	for i, p := range deviceData.Values {
		point, ok := CoreCache.GetPointByDevice(deviceData.ID, p.PointName)
		if !ok {
			//todo 临时屏蔽vrf异常日志输出
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				Logger.Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", p.PointName))
			}
			continue
		}
		//点位值类型还原
		value, err := ConvPointType(p.Value, point.ValueType)
		if err != nil {
			Logger.Error("convert point value error", zap.Error(err), zap.Any("deviceId", deviceData.ID),
				zap.String("pointName", p.PointName), zap.Any("value", p.Value))
			continue
		}

		//精度换算
		if !driverEnable && point.Scale != 0 {
			value, err = multiplyWithFloat64(value, point.Scale)
			if err != nil {
				Logger.Error("multiplyWithFloat64 error", zap.Error(err), zap.Any("deviceId", deviceData.ID))
				continue
			}
		}

		//浮点类型,且readValue包含小数时作小数保留位数加工
		if point.ValueType == config.ValueType_Float && value != 0 {
			val := fmt.Sprintf("%.*f", point.Decimals, value)
			value, _ = strconv.ParseFloat(val, 64)
		}
		deviceData.Values[i].Value = value
	}
	return nil
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
