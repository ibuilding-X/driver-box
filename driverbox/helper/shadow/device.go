package shadow

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"log"
	"time"
)

// Device 设备结构
type Device struct {
	name            string                 // 设备名称
	modelName       string                 // 设备模型名称
	points          map[string]devicePoint // 设备点位列表
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

// devicePoint 设备点位结构
type devicePoint struct {
	Name      string      // 点位名称
	Value     interface{} // 点位值
	UpdatedAt time.Time   // 点位最后更新时间（用于点位缓存过期判断）
}

func NewDevice(device config.DeviceBase, modelName string, points map[string]devicePoint) Device {
	//默认24小时无数据上报，视为设备离线
	ttl := time.Duration(24) * time.Hour
	if device.Ttl != "" {
		t, err := time.ParseDuration(device.Ttl)
		if err != nil {
			log.Fatalf("device:%v parse ttl:%v error:%v", device.Name, device.Ttl, err)
		} else {
			ttl = t
		}
	} else {
		log.Printf("device:%v ttl unset, reset default value:%v", device.Name, ttl)
	}
	return Device{
		name:            device.Name,
		modelName:       modelName,
		points:          points,
		onlineBindPoint: "",
		ttl:             ttl,
		online:          false, // 默认设备处于离线状态
	}
}
