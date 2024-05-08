package core

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

// SendBatchWrite 发送多个点位写命令
func SendBatchWrite(deviceId string, points []plugin.PointData) (err error) {
	var connector plugin.Connector
	defer func() {
		// 释放连接
		if connector != nil {
			connector.Release()
		}
	}()
	//todo 设备层驱动
	result, driverFlag := deviceDriverProcess(deviceId, plugin.WriteMode, points...)

	if driverFlag && result.Error != nil {
		if result.Error != nil {
			return result.Error
		} else {
			points = result.Points
		}
	}

	//按照点位的协议、连接分组
	for _, pd := range points {
		point, ok := helper.CoreCache.GetPointByDevice(deviceId, pd.PointName)
		if !ok {
			return fmt.Errorf("not found point, point name is %s", pd.PointName)
		}
		//未进设备驱动加工，执行数值精度换算逻辑
		if !driverFlag {
			err := pointScaleProcess(&pd, point)
			if err != nil {
				return err
			}
		}

		// 获取连接
		if connector != nil {
			continue
		}
		p, ok := helper.CoreCache.GetRunningPluginByDeviceAndPoint(deviceId, pd.PointName)
		if !ok {
			return fmt.Errorf("not found running plugin, device name is %s", deviceId)
		}
		connector, err = p.Connector(deviceId, pd.PointName)
		if err != nil {
			_ = helper.DeviceShadow.MayBeOffline(deviceId)
			return err
		}
	}
	//按连接批量下发
	adapter := connector.ProtocolAdapter()
	res, err := adapter.Encode(deviceId, plugin.WriteMode, points...)
	if err != nil {
		return err
	}
	// 发送数据
	if err = connector.Send(res); err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceId)
		return err
	}
	//点位写成功后，立即触发读取操作以及时更新影子状态
	for _, pointData := range points {
		tryReadNewValue(deviceId, pointData.PointName, pointData.Value)
	}
	return
}
