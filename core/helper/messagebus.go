package helper

import (
	"driver-box/core/contracts"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"go.uber.org/zap"
)

// WriteToMessageBus 设备数据写入消息总线
func WriteToMessageBus(deviceData contracts.DeviceData) {
	var values []*sdkModels.CommandValue
	for _, point := range deviceData.Values {
		// 获取点位信息
		cachePoint, ok := CoreCache.GetPointByDevice(deviceData.DeviceName, point.PointName)
		if !ok {
			Logger.Warn("unknown point", zap.Any("deviceName", deviceData.DeviceName), zap.Any("pointName", point.PointName))
			continue
		}
		// 缓存比较
		shadowValue, _ := DeviceShadow.GetDevicePoint(deviceData.DeviceName, point.PointName)
		if shadowValue == point.Value {
			Logger.Debug("point value = cache, stop sending to messageBus")
			continue
		}
		// 缓存
		if err := DeviceShadow.SetDevicePoint(deviceData.DeviceName, point.PointName, point.Value); err != nil {
			Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceName", deviceData.DeviceName))
		}
		// 点位类型转换
		pointValue, err := ConvPointType(point.Value, cachePoint.ValueType)
		if err != nil {
			Logger.Warn("point value type convert error", zap.Error(err))
			continue
		}
		// 点位值类型名称转换
		pointType := PointValueType2EdgeX(cachePoint.ValueType)
		v, err := sdkModels.NewCommandValue(point.PointName, pointType, pointValue)
		if err != nil {
			Logger.Warn("new command value error", zap.Error(err), zap.Any("pointName", point.PointName), zap.Any("type", pointType), zap.Any("value", pointValue))
			continue
		}
		values = append(values, v)
	}
	if len(values) > 0 {
		Logger.Info("send to message bus", zap.Any("deviceName", deviceData.DeviceName), zap.Any("values", values))
		MessageBus <- &sdkModels.AsyncValues{
			DeviceName:    deviceData.DeviceName,
			SourceName:    "default",
			CommandValues: values,
		}
	}
}
