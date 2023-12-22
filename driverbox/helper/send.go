package helper

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

// Send 向设备发送数据
func Send(deviceName string, mode plugin.EncodeMode, value plugin.PointData) (err error) {
	defer func() {
		if err2 := recover(); err2 != nil {
			Logger.Error(fmt.Sprintf("%+v", err2))
		}
	}()
	value.Value, err = ConvPointType(value.Value, value.Type)
	if err != nil {
		return err
	}
	// 获取插件
	p, ok := CoreCache.GetRunningPluginByDeviceAndPoint(deviceName, value.PointName)
	if !ok {
		return fmt.Errorf("not found running plugin, device name is %s", deviceName)
	}
	// 获取连接
	conn, err := p.Connector(deviceName, value.PointName)
	if err != nil {
		_ = DeviceShadow.MayBeOffline(deviceName)
		return
	}
	// 释放连接
	defer conn.Release()
	// 协议适配器
	adapter := p.ProtocolAdapter()
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
			err2 := Send(deviceName, plugin.ReadMode, plugin.PointData{
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
