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
func SendBatchWrite(deviceSn string, points []plugin.PointData) (err error) {
	var connector plugin.Connector
	defer func() {
		// 释放连接
		if connector != nil {
			connector.Release()
		}
	}()
	//按照点位的协议、连接分组
	for _, pd := range points {
		point, ok := helper.CoreCache.GetPointByDevice(deviceSn, pd.PointName)
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
		p, ok := helper.CoreCache.GetRunningPluginByDeviceAndPoint(deviceSn, pd.PointName)
		if !ok {
			return fmt.Errorf("not found running plugin, device name is %s", deviceSn)
		}
		connector, err = p.Connector(deviceSn, pd.PointName)
		if err != nil {
			_ = helper.DeviceShadow.MayBeOffline(deviceSn)
			return err
		}
	}
	//按连接批量下发
	adapter := connector.ProtocolAdapter()
	res, err := adapter.Encode(deviceSn, plugin.WriteMode, points...)
	if err != nil {
		return err
	}
	// 发送数据
	if err = connector.Send(res); err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceSn)
		return err
	}
	//点位写成功后，立即触发读取操作以及时更新影子状态
	for _, pointData := range points {
		tryReadNewValue(deviceSn, pointData.PointName, pointData.Value)
	}
	return
}
