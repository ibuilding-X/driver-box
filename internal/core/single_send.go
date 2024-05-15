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

// 单点操作
func SendSinglePoint(deviceId string, mode plugin.EncodeMode, pointData plugin.PointData) error {
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
				tryReadNewValue(deviceId, pointData.PointName, pointData.Value)
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

// 尝试读取期望点位值
func tryReadNewValue(deviceId, pointName string, expectValue interface{}) {
	point, ok := helper.CoreCache.GetPointByDevice(deviceId, pointName)
	if !ok {
		return
	}
	if point.ReadWrite != config.ReadWrite_R && point.ReadWrite != config.ReadWrite_RW {
		return
	}
	//延迟100毫秒触发读操作
	go func(deviceId, pointName string, expectValue interface{}) {
		i := 0
		for i < 10 {
			i++
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			helper.Logger.Info("point write success,try to read new value", zap.String("point", pointName))
			err := SendSinglePoint(deviceId, plugin.ReadMode, plugin.PointData{
				PointName: pointName,
			})
			if err != nil {
				helper.Logger.Error("point write success, read new value error", zap.String("point", pointName), zap.Error(err))
				break
			}

			value, _ := helper.DeviceShadow.GetDevicePoint(deviceId, pointName)
			helper.Logger.Info("point write success, read new value", zap.String("point", pointName), zap.Any("expect", expectValue), zap.Any("value", value))
			if fmt.Sprint(expectValue) == fmt.Sprint(value) {
				break
			}
		}
	}(deviceId, pointName, expectValue)
}
