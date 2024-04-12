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
func SendSinglePoint(deviceSn string, mode plugin.EncodeMode, pointData plugin.PointData) error {
	point, ok := helper.CoreCache.GetPointByDevice(deviceSn, pointData.PointName)
	if !ok {
		return fmt.Errorf("not found point, point name is %s", pointData.PointName)
	}

	//精度换算
	if mode == plugin.WriteMode {
		err := pointValueProcess(&pointData, point)
		if err != nil {
			return err
		}
	}

	//判断点位操作有效性
	if mode == plugin.WriteMode && point.ReadWrite == config.ReadWrite_R {
		return errors.New("point is readonly, can not write")
	} else if mode == plugin.ReadMode && point.ReadWrite == config.ReadWrite_W {
		return errors.New("point is writeOnly, can not read")
	}

	// 获取插件
	p, ok := helper.CoreCache.GetRunningPluginByDeviceAndPoint(deviceSn, pointData.PointName)
	if !ok {
		return fmt.Errorf("not found running plugin, device name is %s", deviceSn)
	}
	// 获取连接
	conn, err := p.Connector(deviceSn, pointData.PointName)
	if err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceSn)
		return err
	}
	// 释放连接
	defer conn.Release()
	// 协议适配器
	adapter := p.ProtocolAdapter()
	res, err := adapter.Encode(deviceSn, mode, pointData)
	if err != nil {
		return err
	}
	// 发送数据
	if err = conn.Send(res); err != nil {
		_ = helper.DeviceShadow.MayBeOffline(deviceSn)
		return err
	}
	//点位写成功后，立即触发读取操作以及时更新影子状态
	if mode == plugin.WriteMode {
		tryReadNewValue(deviceSn, pointData.PointName, pointData.Value)
	}
	return err
}

func pointValueProcess(pointData *plugin.PointData, point config.Point) error {
	if point.Scale == 0 {
		return nil
	}
	value, err := helper.ConvPointType(pointData.Value, point.ValueType)
	if err != nil {
		return err
	}
	if point.Scale != 0 {
		value, err = divideStrings(value, point.Scale)
		if err != nil {
			return err
		}
	}
	pointData.Value = value
	return nil
}

// 尝试读取期望点位值
func tryReadNewValue(deviceSn, pointName string, expectValue interface{}) {
	point, ok := helper.CoreCache.GetPointByDevice(deviceSn, pointName)
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
			helper.Logger.Info("point write success,try to read new value", zap.String("point", pointName))
			err := SendSinglePoint(deviceSn, plugin.ReadMode, plugin.PointData{
				PointName: pointName,
			})
			if err != nil {
				helper.Logger.Error("point write success, read new value error", zap.String("point", pointName), zap.Error(err))
				break
			}

			value, _ := helper.DeviceShadow.GetDevicePoint(deviceSn, pointName)
			helper.Logger.Info("point write success, read new value", zap.String("point", pointName), zap.Any("expect", expectValue), zap.Any("value", value))
			if fmt.Sprint(expectValue) == fmt.Sprint(value) {
				break
			}
		}
	}(deviceSn, pointName, expectValue)
}

func divideStrings(value interface{}, scale float64) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v / scale, nil
	case int64:
		return float64(v) / scale, nil
	default:
		return 0, fmt.Errorf("cannot divide %T with float64", value)
	}
}
