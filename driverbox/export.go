package driverbox

import (
	"fmt"
	"math"
	"strings"

	"github.com/ibuilding-x/driver-box/driverbox/export"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	export0 "github.com/ibuilding-x/driver-box/internal/export"
	"github.com/ibuilding-x/driver-box/internal/export/base"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"github.com/ibuilding-x/driver-box/pkg/library"
	"go.uber.org/zap"
)

func init() {
	EnableExport(base.Get())
}

// EnableExport 注册Export至driver-box
// 该函数将实现了Export接口的模块注册到系统中
// 如果相同的Export实例已存在，则不会重复注册
// 参数:
//   - export: 实现了export.Export接口的导出模块实例
//
// 使用示例:
//
//	type CustomExport struct{}
//
//	func (c *CustomExport) Init() error { return nil }
//	func (c *CustomExport) ExportTo(deviceData plugin.DeviceData) {}
//	func (c *CustomExport) OnEvent(eventCode event.EventCode, key string, eventValue interface{}) error { return nil }
//	func (c *CustomExport) IsReady() bool { return true }
//	func (c *CustomExport) Destroy() error { return nil }
//
//	customExport := &CustomExport{}
//	driverbox.EnableExport(customExport)
func EnableExport(export export.Export) {
	for _, e := range export0.Exports {
		if e == export {
			return
		}
	}
	export0.Exports = append(export0.Exports, export)
}

// TriggerEvents 触发事件通知到所有已加载的Export插件
// 参数:
//   - eventCode: 事件代码，标识事件类型
//     常见事件类型: event.DeviceDiscover(设备发现), event.SceneTrigger(场景触发),
//     event.ServiceStatus(服务状态变更), event.Exporting(导出前事件)等
//   - key: 事件关联的键值，通常是设备ID、服务序列号或其他唯一标识
//   - value: 事件携带的数据值，可以是任意类型的数据
//
// 功能:
//
//	遍历所有已加载的Export插件，调用其OnEvent方法处理事件
//	如果Export插件未就绪，则跳过该插件
//	记录事件处理过程中的错误信息
//
// 注意事项:
//   - 此函数会并发调用各个Export插件的OnEvent方法
//   - 单个插件处理失败不影响其他插件的处理
func TriggerEvents(eventCode event.EventCode, key string, value interface{}) {
	export0.TriggerEvents(eventCode, key, value)
}

// Export 导出设备数据到各个Export插件
// 这是设备数据导出的核心函数，负责将设备数据分发到所有已注册的Export模块
// 参数:
//   - deviceData: 设备数据数组，包含设备ID、数值、事件等信息
//     每个DeviceData包含: ID(设备ID), Values(点位值数组), Events(事件数组), ExportType(导出类型)
//
// 处理流程:
//  1. 记录调试日志
//  2. 触发插件回调事件(event.DoExport)
//  3. 遍历每个设备数据:
//     - 如果设备有事件，则触发事件通知
//     - 对点位数据进行缓存过滤(根据报告模式过滤不变的点位值)
//     - 如果设备没有数值数据则跳过
//     - 将数据导出到所有已准备好的Export插件
//
// 数据过滤机制:
//   - 根据报告模式过滤不变的点位值(ReportMode_Change)
//   - 缓存点位值以检测变化
//   - 触发预处理事件(event.Exporting)
func Export(deviceData []plugin.DeviceData) {
	Log().Debug("export data", zap.Any("data", deviceData))
	// 产生插件回调事件
	TriggerEvents(event.DoExport, "", deviceData)
	// 写入消息总线
	for _, data := range deviceData {
		//触发事件通知
		if len(data.Events) > 0 {
			for _, evt := range data.Events {
				TriggerEvents(event.EventCode(evt.Code), data.ID, evt.Value)
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

// BaseExport 获取基础导出模块实例
// 基础导出模块提供标准的API接口和其他基本导出功能
// 返回值:
//   - base.BaseExport: 基础导出模块实例，可用来注册自定义API、配置服务等
//
// 使用示例:
//
//	baseExport := driverbox.BaseExport()
//	// 注册自定义API接口
//	// baseExport.RegisterAPI("/custom/api", handler)
//	// 配置HTTP服务器参数
//	// baseExport.SetHTTPConfig(config)
func BaseExport() base.BaseExport {
	return base.Get()
}

// pointCacheFilter 对设备数据进行缓存过滤处理
// 该函数执行以下操作:
// 1. 对点位值进行预处理(包括驱动解码和类型转换)
// 2. 根据报告模式过滤点位值(只传递变化的值)
// 3. 缓存点位值到设备影子服务
// 4. 触发预导出事件
// 参数:
//   - deviceData: 指向设备数据的指针，函数会直接修改该数据
//
// 过滤逻辑:
//   - 如果点位报告模式为ReportMode_Change且值未变化，则过滤掉该点位
//   - 将处理后的数据缓存到设备影子中
//   - 触发event.Exporting事件供其他模块处理
func pointCacheFilter(deviceData *plugin.DeviceData) {
	// 设备层驱动，对点位进行预处理
	err := pointValueProcess(deviceData)
	if err != nil {
		Log().Error("device driver process error", zap.Any("deviceData", deviceData), zap.Error(err))
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
				Log().Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", point.PointName))
			}
			continue
		}

		// 缓存比较
		shadowValue, _ := Shadow().GetDevicePoint(deviceData.ID, point.PointName)

		// 如果是周期上报模式，且缓存中有值，停止触发
		if p.ReportMode() == config.ReportMode_Change && shadowValue == point.Value {
			Log().Debug("point report mode is change, stop to trigger Export", zap.String("pointName", p.Name()))
		} else {
			// 点位值类型名称转换
			points = append(points, point)
		}

		// 缓存
		if err := Shadow().SetDevicePoint(deviceData.ID, point.PointName, point.Value); err != nil {
			Log().Error("shadow store point value error", zap.Error(err), zap.Any("deviceId", deviceData.ID))
		}
	}
	deviceData.Values = points
	deviceData.ExportType = plugin.RealTimeExport

	TriggerEvents(event.Exporting, deviceData.ID, originalData)
}

// pointValueProcess 对点位值进行预处理
// 该函数执行以下处理:
// 1. 通过设备驱动库对点位值进行解码和加工
// 2. 执行点位值类型转换
// 3. 应用精度换算(scale)
// 4. 应用小数位数保留
// 参数:
//   - deviceData: 指向设备数据的指针，函数会直接修改其中的值
//
// 返回值:
//   - error: 处理过程中发生的错误
//
// 处理流程:
//   - 检查设备是否存在及是否有驱动配置
//   - 使用设备驱动对点位值进行解码处理
//   - 执行类型转换确保值符合点位定义
//   - 应用缩放因子调整数值精度
//   - 应用小数位数限制
func pointValueProcess(deviceData *plugin.DeviceData) error {
	device, ok := CoreCache().GetDevice(deviceData.ID)
	if !ok {
		Log().Error("unknown device", zap.Any("deviceId", deviceData.ID))
		return fmt.Errorf("unknown device")
	}
	driverEnable := len(device.DriverKey) > 0

	//通过设备层驱动对点位值进行加工
	if driverEnable {
		result := library.Driver().DeviceDecode(device.DriverKey, library.DeviceDecodeRequest{DeviceId: deviceData.ID, Points: deviceData.Values})
		if result.Error != nil {
			Log().Error("library.DeviceDecode error", zap.Error(result.Error), zap.Any("deviceData", deviceData))
			return result.Error
		} else {
			deviceData.Values = result.Points
			//驱动产生的事件
			if len(result.Events) > 0 {
				for _, e := range result.Events {
					TriggerEvents(event.EventCode(e.Code), deviceData.ID, e.Value)
				}
			}
		}
	}
	for i, p := range deviceData.Values {
		point, ok := CoreCache().GetPointByDevice(deviceData.ID, p.PointName)
		if !ok {
			//todo 临时屏蔽vrf异常日志输出
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				Log().Error("unknown point", zap.Any("deviceId", deviceData.ID), zap.Any("pointName", p.PointName))
			}
			continue
		}
		//点位值类型还原
		value, err := convutil.PointValue(p.Value, point.ValueType())
		if err != nil {
			if !strings.HasPrefix(deviceData.ID, "vrf/") {
				Log().Error("convert point value error", zap.Error(err), zap.Any("deviceId", deviceData.ID),
					zap.String("pointName", p.PointName), zap.Any("value", p.Value))
			}
			continue
		}

		//精度换算
		if !driverEnable && point.Scale() != 0 {
			value, err = multiplyWithFloat64(value, point.Scale())
			if err != nil {
				Log().Error("multiplyWithFloat64 error", zap.Error(err), zap.Any("deviceId", deviceData.ID))
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

// multiplyWithFloat64 将任意数值类型与浮点数相乘
// 该函数处理各种数值类型的乘法运算，并将其结果转换为float64
// 参数:
//   - value: 需要相乘的值，支持多种数值类型(int, float, uint等)
//   - scale: 乘数，浮点数，通常用于精度换算
//
// 返回值:
//   - float64: 相乘结果，统一为float64类型
//   - error: 类型不支持时返回错误
//
// 支持的类型:
//   - 整数类型: int, int16, int32, int64, uint16, uint32, uint64
//   - 浮点类型: float32, float64
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
