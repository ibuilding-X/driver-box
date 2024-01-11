package helper

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

// Send 向设备发送数据
func Send(deviceSn string, mode plugin.EncodeMode, value plugin.PointData) (err error) {
	defer func() {
		if err2 := recover(); err2 != nil {
			Logger.Error(fmt.Sprintf("%+v", err2))
		}
	}()
	point, ok := CoreCache.GetPointByDevice(deviceSn, value.PointName)
	if !ok {
		return fmt.Errorf("not found point, point name is %s", value.PointName)
	}
	value.Value, err = ConvPointType(value.Value, point.ValueType)
	if err != nil {
		return err
	}
	// 获取插件
	p, ok := CoreCache.GetRunningPluginByDeviceAndPoint(deviceSn, value.PointName)
	if !ok {
		return fmt.Errorf("not found running plugin, device name is %s", deviceSn)
	}
	// 获取连接
	conn, err := p.Connector(deviceSn, value.PointName)
	if err != nil {
		_ = DeviceShadow.MayBeOffline(deviceSn)
		return
	}
	// 释放连接
	defer conn.Release()
	// 协议适配器
	adapter := p.ProtocolAdapter()
	res, err := adapter.Encode(deviceSn, mode, value)
	if err != nil {
		return
	}
	// 发送数据
	if err = conn.Send(res); err != nil {
		_ = DeviceShadow.MayBeOffline(deviceSn)
		return
	}
	return
}

// SendMultiRead 发送多个点位读取命令，多用于 autoEvent
func SendMultiRead(devicesSn []string, pointNames []string) (err error) {
	for i, _ := range devicesSn {
		deviceSn := devicesSn[i]
		for _, pointName := range pointNames {
			err2 := Send(deviceSn, plugin.ReadMode, plugin.PointData{
				PointName: pointName,
			})
			if err2 != nil {
				Logger.Error(fmt.Sprintf("send error: %s", err2.Error()))
			}
		}
	}

	return
}
