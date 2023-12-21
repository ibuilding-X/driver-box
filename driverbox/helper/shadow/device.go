package shadow

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"go.uber.org/zap"
	"time"
)

// Device 设备结构
type Device struct {
	Name            string                 // 设备名称
	ModelName       string                 // 设备模型名称
	Points          map[string]DevicePoint // 设备点位列表
	onlineBindPoint string                 // 在线状态绑定点位（支持数据类型：bool、string、int、float）
	online          bool                   // 在线状态
	ttl             time.Duration          // 设备离线阈值，超过该时长没有收到数据视为离线
	disconnectTimes int                    // 断开连接次数，60秒内超过3次判定离线
	updatedAt       time.Time              // 更新时间（用于设备离线判断）
}

// SetOnlineBindPoint 设备在线状态绑定指定点位
func (d *Device) SetOnlineBindPoint(pointName string) {
	d.onlineBindPoint = pointName
}

// DevicePoint 设备点位结构
type DevicePoint struct {
	Name      string      // 点位名称
	Value     interface{} // 点位值
	UpdatedAt time.Time   // 点位最后更新时间（用于点位缓存过期判断）
}

func NewDevice(device config.DeviceBase, modelName string, points map[string]DevicePoint) Device {
	//默认24小时无数据上报，视为设备离线
	ttl := time.Duration(24) * time.Hour
	if device.Ttl != "" {
		t, err := time.ParseDuration(device.Ttl)
		if err != nil {
			helper.Logger.Error("parse ttl error", zap.String("device", device.Name), zap.String("ttl", device.Ttl), zap.Error(err))
		} else {
			ttl = t
		}
	} else {
		helper.Logger.Info("device ttl unset, reset default value", zap.String("device", device.Name), zap.Any("ttl", ttl))
	}
	return Device{
		Name:            device.Name,
		ModelName:       modelName,
		Points:          points,
		onlineBindPoint: "",
		ttl:             ttl,
		online:          false, // 默认设备处于离线状态
	}
}

func NewDevicePoint(pointName string, value interface{}) DevicePoint {
	return DevicePoint{
		Name:      pointName,
		Value:     value,
		UpdatedAt: time.Now(),
	}
}
