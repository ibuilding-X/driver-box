package core

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
)

type batchWrite struct {
	points    []plugin.PointData
	connector plugin.Plugin
}

// SendBatchWrite 发送多个点位写命令
func SendBatchWrite(deviceId string, points []plugin.PointData) (err error) {
	var connector plugin.Connector
	defer func() {
		// 释放连接
		if connector != nil {
			connector.Release()
		}
	}()
	//按照点位的协议、连接分组
	for _, pd := range points {
		point, ok := helper.CoreCache.GetPointByDevice(deviceId, pd.PointName)
		if !ok {
			return fmt.Errorf("not found point, point name is %s", pd.PointName)
		}
		//数值精度换算
		err := pointValueProcess(&pd, point)
		if err != nil {
			return err
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
