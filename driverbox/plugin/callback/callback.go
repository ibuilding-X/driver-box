package callback

import (
	"fmt"
	"math"
	"strings"

	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/utils"
	"github.com/ibuilding-x/driver-box/driverbox/library"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/export"
	"go.uber.org/zap"
)

func ExportTo(deviceData []plugin.DeviceData) {
	helper.Logger.Debug("export data", zap.Any("data", deviceData))
	// 产生插件回调事件
	export.TriggerEvents(event.EventCodePluginCallback, "", deviceData)
	// 写入消息总线
	for _, data := range deviceData {
		//触发事件通知
		if len(data.Events) > 0 {
			for _, event := range data.Events {
				export.TriggerEvents(event.Code, data.ID, event.Value)
			}
		}
		pointCacheFilter(&data)
		if len(data.Values) == 0 {
			continue
		}
		for _, export := range export.Exports {
			if export.IsReady() {
				export.ExportTo(data)
			}
		}
	}
}
func pointCacheFilter(deviceData *plugin.DeviceData) {
	// 设备层驱动，对点位进行预处理
	err := pointValueProcess(deviceData)
	if err != nil {
		helper.Logger.Error("device driver process error", zap.Any("deviceData", deviceData), zap.Error(err))
		return
	}
	//获取完成点位加工后的真实 deviceData
	originalData := plugin.DeviceData{
		ID:         deviceData.ID,
		Values:     deviceData.Values,
		Events:     deviceData.Events,
		ExportType: deviceData.ExportType,
	}

	// 定义一个空的整型数组
	var points []plugin.PointData
	for _, point := range deviceData.Values {
		// 获取点位信息
		p, ok := helper.CoreCache.GetPointByDevice(deviceData.ID, point.PointName)
		if !ok {
			//todo 临时屏蔽vrf异常日志输出
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				helper.Logger.Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", point.PointName))
			}
			continue
		}

		// 缓存比较
		shadowValue, _ := helper.DeviceShadow.GetDevicePoint(deviceData.ID, point.PointName)

		// 如果是周期上报模式，且缓存中有值，停止触发
		if p.ReportMode() == config.ReportMode_Change && shadowValue == point.Value {
			helper.Logger.Debug("point report mode is change, stop to trigger ExportTo", zap.String("pointName", p.Name()))
		} else {
			// 点位值类型名称转换
			points = append(points, point)
		}

		// 缓存
		if err := helper.DeviceShadow.SetDevicePoint(deviceData.ID, point.PointName, point.Value); err != nil {
			helper.Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceId", deviceData.ID))
		}
	}
	deviceData.Values = points
	deviceData.ExportType = plugin.RealTimeExport

	export.TriggerEvents(event.EventCodeWillExportTo, deviceData.ID, originalData)
}

func pointValueProcess(deviceData *plugin.DeviceData) error {
	device, ok := helper.CoreCache.GetDevice(deviceData.ID)
	if !ok {
		helper.Logger.Error("unknown device", zap.Any("deviceId", deviceData.ID))
		return fmt.Errorf("unknown device")
	}
	driverEnable := len(device.DriverKey) > 0

	//通过设备层驱动对点位值进行加工
	if driverEnable {
		result := library.Driver().DeviceDecode(device.DriverKey, library.DeviceDecodeRequest{DeviceId: deviceData.ID, Points: deviceData.Values})
		if result.Error != nil {
			helper.Logger.Error("library.DeviceDecode error", zap.Error(result.Error), zap.Any("deviceData", deviceData))
			return result.Error
		} else {
			deviceData.Values = result.Points
			//驱动产生的事件
			if len(result.Events) > 0 {
				for _, e := range result.Events {
					export.TriggerEvents(e.Code, deviceData.ID, e.Value)
				}
			}
		}
	}
	for i, p := range deviceData.Values {
		point, ok := helper.CoreCache.GetPointByDevice(deviceData.ID, p.PointName)
		if !ok {
			//todo 临时屏蔽vrf异常日志输出
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				helper.Logger.Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", p.PointName))
			}
			continue
		}
		//点位值类型还原
		value, err := utils.ConvPointType(p.Value, point.ValueType())
		if err != nil {
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				helper.Logger.Error("convert point value error", zap.Error(err), zap.Any("deviceId", deviceData.ID),
					zap.String("pointName", p.PointName), zap.Any("value", p.Value))
			}
			continue
		}

		//精度换算
		if !driverEnable && point.Scale() != 0 {
			value, err = multiplyWithFloat64(value, point.Scale())
			if err != nil {
				helper.Logger.Error("multiplyWithFloat64 error", zap.Error(err), zap.Any("deviceId", deviceData.ID))
				continue
			}
		}

		//浮点类型,且readValue包含小数时作小数保留位数加工
		if point.ValueType() == config.ValueType_Float && value != 0 {
			multiplier := math.Pow(10, float64(point.Decimals()))
			// 先转成整数，再通过除法实现小数位数保留
			value = math.Trunc(value.(float64)*multiplier) / multiplier
		}
		deviceData.Values[i].Value = value
	}
	return nil
}

func multiplyWithFloat64(value interface{}, scale float64) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v * scale, nil
	case int:
		return float64(v) * scale, nil
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
