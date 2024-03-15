package shadow

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"log"
	"sync"
	"time"
)

// Device 设备结构
type Device struct {
	deviceSn        string        // 设备SN
	modelName       string        // 设备模型名称
	points          *sync.Map     // 设备点位列表
	onlineBindPoint string        // 在线状态绑定点位（支持数据类型：bool、string、int、float）
	online          bool          // 在线状态
	ttl             time.Duration // 设备离线阈值，超过该时长没有收到数据视为离线
	disconnectTimes int           // 断开连接次数，60秒内超过3次判定离线
	updatedAt       time.Time     // 更新时间（用于设备离线判断）
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

func NewDevice(device config.Device, modelName string, points map[string]DevicePoint) Device {
	//默认24小时无数据上报，视为设备离线
	ttl := time.Duration(24) * time.Hour
	if device.Ttl != "" {
		t, err := time.ParseDuration(device.Ttl)
		if err != nil {
			log.Fatalf("device:%v parse ttl:%v error:%v", device.DeviceSn, device.Ttl, err)
		} else {
			ttl = t
		}
	} else {
		log.Printf("device:%v ttl unset, reset default value:%v", device.DeviceSn, ttl)
	}
	// 转换 points
	ps := &sync.Map{}
	for k, _ := range points {
		ps.Store(k, points[k])
	}
	return Device{
		deviceSn:        device.DeviceSn,
		modelName:       modelName,
		points:          ps,
		onlineBindPoint: "",
		ttl:             ttl,
		online:          false, // 默认设备处于离线状态
	}
}
