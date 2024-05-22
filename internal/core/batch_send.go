package core

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"time"
)

// SendBatchWrite 发送多个点位写命令
func SendBatchWrite(deviceId string, points []plugin.PointData) (err error) {
	//设备驱动层加工
	result, err := deviceDriverProcess(deviceId, plugin.WriteMode, points...)
	if err != nil {
		return err
	}
	if len(result) == 0 {
		helper.Logger.Warn("device driver process result is empty", zap.String("deviceId", deviceId))
		return nil
	}

	connector, err := getConnector(deviceId, result)
	defer func() {
		// 释放连接
		if connector != nil {
			_ = connector.Release()
		}
	}()
	if err != nil {
		return err
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
	tryReadNewValues(deviceId, points)
	return
}

// 批量读取多个点位值
func SendBatchRead(deviceId string, points []plugin.PointData) (err error) {
	readPoints := make([]plugin.PointData, 0)
	for _, p := range points {
		point, ok := helper.CoreCache.GetPointByDevice(deviceId, p.PointName)
		if !ok {
			return errors.New("point not found")
		}
		if point.ReadWrite != config.ReadWrite_R && point.ReadWrite != config.ReadWrite_RW {
			return errors.New("point is not readable")
		}
		readPoints = append(readPoints, p)
	}
	if len(readPoints) == 0 {
		return
	}

	connector, err := getConnector(deviceId, readPoints)
	defer func() {
		// 释放连接
		if connector != nil {
			_ = connector.Release()
		}
	}()
	if err != nil {
		return err
	}
	//按连接批量下发
	adapter := connector.ProtocolAdapter()
	res, err := adapter.Encode(deviceId, plugin.ReadMode, points...)
	if err != nil {
		return err
	}
	// 发送数据
	if err = connector.Send(res); err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceId)
		return err
	}
	return nil
}

// 尝试读取期望点位值
func tryReadNewValues(deviceId string, points []plugin.PointData) {
	readPoints := make([]plugin.PointData, 0)
	for _, p := range points {
		point, ok := helper.CoreCache.GetPointByDevice(deviceId, p.PointName)
		if !ok {
			return
		}
		if point.ReadWrite != config.ReadWrite_R && point.ReadWrite != config.ReadWrite_RW {
			continue
		}
		readPoints = append(readPoints, p)
	}
	//延迟100毫秒触发读操作
	go func(deviceId string, readPoints []plugin.PointData) {
		i := 0
		for i < 10 {
			i++
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			helper.Logger.Info("point write success,try to read new value", zap.Any("points", readPoints))
			err := SendBatchRead(deviceId, readPoints)
			if err != nil {
				helper.Logger.Error("point write success, read new value error", zap.Any("points", readPoints), zap.Error(err))
				break
			}

			ok := true
			for _, p := range readPoints {
				value, _ := helper.DeviceShadow.GetDevicePoint(deviceId, p.PointName)
				helper.Logger.Info("point write success, read new value", zap.String("point", p.PointName), zap.Any("expect", p.Value), zap.Any("value", value))
				if fmt.Sprint(p.Value) != fmt.Sprint(value) {
					ok = false
					break
				}
			}
			if ok {
				break
			}
		}
	}(deviceId, readPoints)
}

func getConnector(deviceId string, points []plugin.PointData) (plugin.Connector, error) {
	//按照点位的协议、连接分组
	for _, pd := range points {
		p, ok := helper.CoreCache.GetRunningPluginByDeviceAndPoint(deviceId, pd.PointName)
		if !ok {
			return nil, fmt.Errorf("not found running plugin, device name is %s", deviceId)
		}
		connector, err := p.Connector(deviceId, pd.PointName)
		if err != nil {
			_ = helper.DeviceShadow.MayBeOffline(deviceId)
			return nil, err
		} else {
			return connector, nil
		}
	}
	return nil, errors.New("not found connector")
}
