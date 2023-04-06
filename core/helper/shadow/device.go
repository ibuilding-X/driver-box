package shadow

import "time"

var defaultDeviceTTL = 15 * time.Minute // 默认设备生命周期

// SetDefaultDeviceTTL 设置默认设备生命周期
func SetDefaultDeviceTTL(ttl time.Duration) {
	defaultDeviceTTL = ttl
}

// Device 设备结构
type Device struct {
	Name            string                 // 设备名称
	ModelName       string                 // 设备模型名称
	Points          map[string]DevicePoint // 设备点位列表
	ttl             time.Duration          // 设备生命周期（即离线判断阈值）
	onlineBindPoint string                 // 在线状态绑定点位（支持数据类型：bool、string、int、float）
	online          bool                   // 在线状态
	disconnectTimes int                    // 断开连接次数，60秒内超过3次判定离线
	updatedAt       time.Time              // 更新时间
}

// SetTTL 设置设备生命周期（超出周期未上报自动判定为离线）
func (d *Device) SetTTL(t time.Duration) {
	d.ttl = t
}

// SetOnlineBindPoint 设备在线状态绑定指定点位
func (d *Device) SetOnlineBindPoint(pointName string) {
	d.onlineBindPoint = pointName
}

// DevicePoint 设备点位结构
type DevicePoint struct {
	Name  string      // 点位名称
	Value interface{} // 点位值
}

func NewDevice(deviceName string, modelName string, points map[string]DevicePoint) Device {
	return Device{
		Name:            deviceName,
		ModelName:       modelName,
		Points:          points,
		ttl:             defaultDeviceTTL,
		onlineBindPoint: "",
		online:          true,
	}
}

func NewDevicePoint(pointName string, value interface{}) DevicePoint {
	return DevicePoint{
		Name:  pointName,
		Value: value,
	}
}
