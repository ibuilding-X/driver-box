package helper

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"time"
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
	//点位写成功后，立即触发读取操作以及时更新影子状态
	if mode == plugin.WriteMode {
		tryReadNewValue(deviceSn, value.PointName, value.Value)
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

// 尝试读取期望点位值
func tryReadNewValue(deviceSn, pointName string, expectValue interface{}) {
	point, ok := CoreCache.GetPointByDevice(deviceSn, pointName)
	if !ok {
		return
	}
	if point.ReadWrite != config.ReadWrite_R && point.ReadWrite != config.ReadWrite_RW {
		return
	}
	//延迟100毫秒触发读操作
	go func(deviceSn, pointName string, expectValue interface{}) {
		i := 0
		for i < 10 {
			i++
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			Logger.Info("point write success,try to read new value", zap.String("point", pointName))
			err := Send(deviceSn, plugin.ReadMode, plugin.PointData{
				PointName: pointName,
			})
			if err != nil {
				Logger.Error("point write success, read new value error", zap.String("point", pointName), zap.Error(err))
				break
			}

			value, _ := DeviceShadow.GetDevicePoint(deviceSn, pointName)
			Logger.Info("point write success, read new value", zap.String("point", pointName), zap.Any("expect", expectValue), zap.Any("value", value))
			if fmt.Sprint(expectValue) == fmt.Sprint(value) {
				break
			}
		}
	}(deviceSn, pointName, expectValue)
}
