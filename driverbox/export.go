package driverbox

import "C"
import (
	"fmt"
	"math"
	"strings"

	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"github.com/ibuilding-x/driver-box/pkg/library"
	"go.uber.org/zap"
)

var Exports exports

// exports 结构体用于管理driver-box框架中的所有Export插件
// 提供加载单个、批量加载以及加载所有内置Export的方法
type exports struct {
}

// LoadExport 加载单个自定义Export插件
// 参数:
//
//	export2: 需要加载的Export插件实例
//
// 功能:
//
//	如果该Export尚未加载，则将其添加到全局Exports列表中
func (exports *exports) LoadExport(export2 export.Export) {
	if !exports.exists(export2) {
		export0.Exports = append(export0.Exports, export2)
	}
}

// LoadExports 批量加载多个Export插件
// 参数:
//
//	export2: 需要加载的Export插件实例数组
//
// 功能:
//
//	遍历数组并调用LoadExport方法逐个加载
func (exports *exports) LoadExports(export2 []export.Export) {
	for _, e := range export2 {
		exports.LoadExport(e)
	}
}

// exists 检查指定的Export是否已经加载
// 参数:
//
//	exp: 需要检查的Export实例
//
// 返回值:
//
//	bool: true表示已加载，false表示未加载
func (exports *exports) exists(exp export.Export) bool {
	for _, e := range export0.Exports {
		if e == exp {
			return true
		}
	}
	return false
}

// 触发事件
// TriggerEvents 触发事件通知到所有已加载的Export插件
// 参数:
//
//	eventCode: 事件代码，标识事件类型
//	key: 事件关联的键值，通常是设备ID或其他标识符
//	value: 事件携带的数据值，可以是任意类型
//
// 功能:
//
//	遍历所有已加载的Export插件，调用其OnEvent方法处理事件
//	如果Export插件未就绪，则跳过该插件
//	记录事件处理过程中的错误信息
func TriggerEvents(eventCode string, key string, value interface{}) {
	export0.TriggerEvents(eventCode, key, value)
}

// Export 导出设备数据到各个Export插件
// 参数:
//
//	deviceData: 设备数据数组，包含设备ID、数值、事件等信息
//
// 功能:
//
//  1. 记录调试日志
//  2. 触发插件回调事件
//  3. 遍历每个设备数据:
//     - 如果设备有事件，则触发事件通知
//     - 对点位数据进行缓存过滤
//     - 如果设备没有数值数据则跳过
//     - 将数据导出到所有已准备好的Export插件
func Export(deviceData []plugin.DeviceData) {
	helper.Logger.Debug("export data", zap.Any("data", deviceData))
	// 产生插件回调事件
	TriggerEvents(event.EventCodePluginCallback, "", deviceData)
	// 写入消息总线
	for _, data := range deviceData {
		//触发事件通知
		if len(data.Events) > 0 {
			for _, event := range data.Events {
				TriggerEvents(event.Code, data.ID, event.Value)
			}
		}
		pointCacheFilter(&data)
		if len(data.Values) == 0 {
			continue
		}
		for _, export := range export0.Exports {
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
		p, ok := CoreCache().GetPointByDevice(deviceData.ID, point.PointName)
		if !ok {
			//todo 临时屏蔽vrf异常日志输出
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				helper.Logger.Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", point.PointName))
			}
			continue
		}

		// 缓存比较
		shadowValue, _ := Shadow().GetDevicePoint(deviceData.ID, point.PointName)

		// 如果是周期上报模式，且缓存中有值，停止触发
		if p.ReportMode() == config.ReportMode_Change && shadowValue == point.Value {
			helper.Logger.Debug("point report mode is change, stop to trigger Export", zap.String("pointName", p.Name()))
		} else {
			// 点位值类型名称转换
			points = append(points, point)
		}

		// 缓存
		if err := Shadow().SetDevicePoint(deviceData.ID, point.PointName, point.Value); err != nil {
			helper.Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceId", deviceData.ID))
		}
	}
	deviceData.Values = points
	deviceData.ExportType = plugin.RealTimeExport

	TriggerEvents(event.EventCodeWillExportTo, deviceData.ID, originalData)
}

func pointValueProcess(deviceData *plugin.DeviceData) error {
	device, ok := CoreCache().GetDevice(deviceData.ID)
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
					TriggerEvents(e.Code, deviceData.ID, e.Value)
				}
			}
		}
	}
	for i, p := range deviceData.Values {
		point, ok := CoreCache().GetPointByDevice(deviceData.ID, p.PointName)
		if !ok {
			//todo 临时屏蔽vrf异常日志输出
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				helper.Logger.Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", p.PointName))
			}
			continue
		}
		//点位值类型还原
		value, err := convutil.PointValue(p.Value, point.ValueType())
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
