package core

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
	"time"
)

// SendBatchWrite 发送多个点位写命令
func SendBatchWrite(deviceId string, points []plugin.PointData) (err error) {
	logger.Logger.Info("send batch write", zap.String("deviceId", deviceId), zap.Any("points", points))
	for _, point := range points {
		_ = helper.DeviceShadow.SetWritePointValue(deviceId, point.PointName, point.Value)
	}
	//设备驱动层加工
	result, err := deviceDriverProcess(deviceId, plugin.WriteMode, points...)
	if err != nil {
		return err
	}
	if len(result) == 0 {
		helper.Logger.Warn("device driver process result is empty", zap.String("deviceId", deviceId))
		return nil
	}

	connector, err := getConnector(deviceId)
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
	res, err := adapter.Encode(deviceId, plugin.WriteMode, result...)
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
	logger.Logger.Info("send batch read", zap.String("deviceId", deviceId), zap.Any("points", points))
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

	connector, err := getConnector(deviceId)
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
	if len(readPoints) == 0 {
		return
	}
	//延迟100毫秒触发读操作
	go func(deviceId string, readPoints []plugin.PointData) {
		checkTime := time.Now()
		i := 0
		for i < 10 {
			i++
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			//优先检查影子状态
			newReadPoints := make([]plugin.PointData, 0)
			shadowPoints, _ := helper.DeviceShadow.GetDevicePoints(deviceId)
			for _, p := range readPoints {
				point, ok := shadowPoints[p.PointName]
				if !ok {
					//设备影子不存在，尝试读取
					newReadPoints = append(newReadPoints, p)
					continue
				}
				if point.WriteAt.After(checkTime) {
					//在checkTime之后有发生过写行为,则本次检验可能不会生效
					helper.Logger.Warn("point write success, but expect point value maybe expired", zap.String("deviceId", deviceId), zap.String("point", p.PointName), zap.Any("expect", p.Value), zap.Any("value", point.Value))
					continue
				}
				if fmt.Sprint(p.Value) != fmt.Sprint(point.Value) {
					newReadPoints = append(newReadPoints, p)
				}
				helper.Logger.Info("point write success, read new value", zap.String("deviceId", deviceId), zap.String("point", p.PointName), zap.Any("expect", p.Value), zap.Any("value", point.Value))
			}
			if len(newReadPoints) == 0 {
				return
			}
			readPoints = newReadPoints

			helper.Logger.Info("point write success,try to read new value", zap.String("deviceId", deviceId), zap.Any("points", readPoints))
			err := SendBatchRead(deviceId, readPoints)
			if err != nil {
				helper.Logger.Error("point write success, read new value error", zap.String("deviceId", deviceId), zap.Any("points", readPoints), zap.Error(err))
				break
			}
		}
	}(deviceId, readPoints)
}

func getConnector(deviceId string) (plugin.Connector, error) {
	//按照点位的协议、连接分组
	p, ok := helper.CoreCache.GetRunningPluginByDevice(deviceId)
	if !ok {
		logger.Logger.Error("not found running plugin", zap.String("deviceId", deviceId))
		return nil, fmt.Errorf("not found running plugin, deviceId: %s ,point: %s", deviceId)
	}
	connector, err := p.Connector(deviceId)
	if err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceId)
		return nil, err
	} else if connector != nil {
		return connector, nil
	} else {
		return nil, errors.New("not found connector")
	}
}
