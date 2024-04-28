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
	group := make(map[plugin.Connector]*batchWrite)
	defer func() {
		// 释放连接
		for c, _ := range group {
			_ = c.Release()
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
		// 获取插件
		p, ok := helper.CoreCache.GetRunningPluginByDeviceAndPoint(deviceSn, pd.PointName)
		if !ok {
			return fmt.Errorf("not found running plugin, device name is %s", deviceSn)
		}
		// 获取连接，相同连接的点位归为一组
		conn, err := p.Connector(deviceSn, pd.PointName)
		if err != nil {
			_ = helper.DeviceShadow.MayBeOffline(deviceSn)
			return err
		}
		bw := group[conn]
		if bw.points == nil {
			bw = &batchWrite{
				points:    make([]plugin.PointData, 0),
				connector: p,
			}
		}
		bw.points = append(bw.points, pd)
	}
	//按连接批量下发
	for conn, bw := range group {
		adapter := conn.ProtocolAdapter()
		res, err := adapter.Encode(deviceSn, plugin.WriteMode, bw.points...)
		if err != nil {
			return err
		}
		// 发送数据
		if err = conn.Send(res); err != nil {
			_ = helper.DeviceShadow.MayBeOffline(deviceSn)
			return err
		}
		//点位写成功后，立即触发读取操作以及时更新影子状态
		for _, pointData := range bw.points {
			tryReadNewValue(deviceSn, pointData.PointName, pointData.Value)
		}
	}
	return
}
