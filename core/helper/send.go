package helper

import (
	"driver-box/core/contracts"
	"fmt"
)

// Send 向设备发送数据
func Send(deviceName string, mode contracts.EncodeMode, value contracts.PointData) (err error) {
	defer func() {
		if err2 := recover(); err2 != nil {
			Logger.Error(fmt.Sprintf("%+v", err2))
		}
	}()
	// 获取插件
	plugin, ok := CoreCache.GetRunningPluginByDevice(deviceName)
	if !ok {
		return fmt.Errorf("not found running plugin, device name is %s", deviceName)
	}
	// 获取连接
	conn, err := plugin.Connector(deviceName)
	if err != nil {
		_ = DeviceShadow.MayBeOffline(deviceName)
		return
	}
	// 释放连接
	defer conn.Release()
	// 协议适配器
	adapter := plugin.ProtocolAdapter()
	res, err := adapter.Encode(deviceName, mode, value)
	if err != nil {
		return
	}
	// 发送数据
	if err = conn.Send(res); err != nil {
		_ = DeviceShadow.MayBeOffline(deviceName)
		return
	}
	return
}

// SendMultiRead 发送多个点位读取命令，多用于 autoEvent
func SendMultiRead(deviceNames []string, pointNames []string) (err error) {
	for i, _ := range deviceNames {
		deviceName := deviceNames[i]
		for _, pointName := range pointNames {
			// 获取点位信息
			point, ok := CoreCache.GetPointByDevice(deviceName, pointName)
			if !ok {
				Logger.Error(fmt.Sprintf("not found point, point name is %s", pointName))
				return
			}
			err2 := Send(deviceName, contracts.ReadMode, contracts.PointData{
				PointName: pointName,
				Type:      point.ValueType,
			})
			if err2 != nil {
				Logger.Error(fmt.Sprintf("send error: %s", err2.Error()))
			}
		}
	}

	return
}
