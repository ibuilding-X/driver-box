package shadow

import "time"

// Device 设备结构
type Device struct {
	Name            string                 // 设备名称
	ModelName       string                 // 设备模型名称
	Points          map[string]DevicePoint // 设备点位列表
	onlineBindPoint string                 // 在线状态绑定点位（支持数据类型：bool、string、int、float）
	online          bool                   // 在线状态
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

func NewDevice(deviceName string, modelName string, points map[string]DevicePoint) Device {
	return Device{
		Name:            deviceName,
		ModelName:       modelName,
		Points:          points,
		onlineBindPoint: "",
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
