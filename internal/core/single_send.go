package core

import (
	"errors"
	"fmt"

	"github.com/ibuilding-x/driver-box/internal/logger"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/config"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	"go.uber.org/zap"
)

// 单点操作
func SendSinglePoint(deviceId string, mode plugin.EncodeMode, pointData plugin.PointData) error {
	logger.Logger.Info("send single point", zap.String("deviceId", deviceId), zap.Any("mode", mode), zap.Any("pointData", pointData))
	_ = helper.DeviceShadow.SetWritePointValue(deviceId, pointData.PointName, pointData.Value)
	if !checkMode(mode) {
		return errors.New("invalid mode")
	}
	result, err := deviceDriverProcess(deviceId, mode, pointData)
	if err != nil {
		return err
	}

	if len(result) == 0 {
		helper.Logger.Warn("device driver process result is empty", zap.String("deviceId", deviceId), zap.Any("mode", mode), zap.Any("pointData", pointData))
		return nil
	}

	for _, v := range result {
		if mode == plugin.WriteMode {
			err = singleWrite(deviceId, v)
			//点位写成功后，立即触发读取操作以及时更新影子状态
			if err == nil {
				tryReadNewValues(deviceId, []plugin.PointData{{
					PointName: pointData.PointName,
					Value:     pointData.Value,
				}})
			}
		} else {
			err = singleRead(deviceId, v)
		}
		if err != nil {
			break
		}
	}
	return err
}

func singleRead(deviceId string, pointData plugin.PointData) error {
	point, ok := helper.CoreCache.GetPointByDevice(deviceId, pointData.PointName)
	if !ok {
		return fmt.Errorf("not found point, point name is %s", pointData.PointName)
	}

	//判断点位操作有效性
	if point.ReadWrite() == config.ReadWrite_W {
		return errors.New("point is writeOnly, can not read")
	}

	// 获取连接
	conn, err := getConnector(deviceId)
	if err != nil {
		return err
	}
	// 释放连接
	defer conn.Release()

	// 协议编码
	res, err := conn.Encode(deviceId, plugin.ReadMode, pointData)
	if err != nil {
		return err
	}
	// 发送数据
	if err = conn.Send(res); err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceId)
		return err
	}

	return err
}

func singleWrite(deviceId string, pointData plugin.PointData) error {
	point, ok := helper.CoreCache.GetPointByDevice(deviceId, pointData.PointName)
	if !ok {
		return fmt.Errorf("not found point, point name is %s", pointData.PointName)
	}

	//判断点位操作有效性
	if point.ReadWrite() == config.ReadWrite_R {
		return errors.New("point is readonly, can not write")
	}

	// 获取连接
	conn, err := getConnector(deviceId)
	if err != nil {
		return err
	}
	// 释放连接
	defer conn.Release()

	// 协议编码
	res, err := conn.Encode(deviceId, plugin.WriteMode, pointData)
	if err != nil {
		return err
	}
	// 发送数据
	if err = conn.Send(res); err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceId)
		return err
	}
	return err
}
