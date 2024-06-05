package core

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"go.uber.org/zap"
)

// 单点操作
func SendSinglePoint(deviceId string, mode plugin.EncodeMode, pointData plugin.PointData) error {
	logger.Logger.Info("send single point", zap.String("deviceId", deviceId), zap.Any("mode", mode), zap.Any("pointData", pointData))
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
	if point.ReadWrite == config.ReadWrite_W {
		return errors.New("point is writeOnly, can not read")
	}

	// 获取插件
	p, ok := helper.CoreCache.GetRunningPluginByDeviceAndPoint(deviceId, pointData.PointName)
	if !ok {
		return fmt.Errorf("not found running plugin, device name is %s", deviceId)
	}
	// 获取连接
	conn, err := p.Connector(deviceId, pointData.PointName)
	if err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceId)
		return err
	}
	// 释放连接
	defer conn.Release()

	// 协议适配器
	adapter := conn.ProtocolAdapter()
	res, err := adapter.Encode(deviceId, plugin.ReadMode, pointData)
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
	if point.ReadWrite == config.ReadWrite_R {
		return errors.New("point is readonly, can not write")
	}

	// 获取插件
	p, ok := helper.CoreCache.GetRunningPluginByDeviceAndPoint(deviceId, pointData.PointName)
	if !ok {
		return fmt.Errorf("not found running plugin, device name is %s", deviceId)
	}
	// 获取连接
	conn, err := p.Connector(deviceId, pointData.PointName)
	if err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceId)
		return err
	}
	// 释放连接
	defer conn.Release()

	// 协议适配器
	adapter := conn.ProtocolAdapter()
	res, err := adapter.Encode(deviceId, plugin.WriteMode, pointData)
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
