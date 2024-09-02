package shadow

import (
	"time"
)

type OnlineChangeCallback func(id string, online bool) // 设备上/下线回调

// Device 设备
type Device struct {
	ID              string                 `json:"id"`
	ModelName       string                 `json:"modelName"`
	Points          map[string]DevicePoint `json:"points"`
	Online          bool                   `json:"online"`
	TTL             string                 `json:"ttl"`
	DisconnectTimes int                    `json:"disconnectTimes"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// DevicePoint 设备点位
type DevicePoint struct {
	Name       string      `json:"name"`
	Value      interface{} `json:"value"`
	WriteValue interface{} `json:"writeValue"`
	UpdatedAt  time.Time   `json:"updatedAt"`
	WriteAt    time.Time   `json:"writeAt"`
}

// DeviceShadow 设备影子
type DeviceShadow interface {
	// AddDevice 新增设备
	AddDevice(id string, modelName string, ttl ...time.Duration)
	// GetDevice 获取设备
	GetDevice(id string) (device Device, ok bool)
	// HasDevice 是否存在设备
	HasDevice(id string) bool
	// DeleteDevice 删除设备
	DeleteDevice(id ...string) (err error)

	// SetDevicePoint 设置设备点位值
	SetDevicePoint(id, pointName string, value interface{}) (err error)
	// GetDevicePoint 获取设备点位值
	GetDevicePoint(id, pointName string) (value interface{}, err error)
	// GetDevicePoints 获取设备所有点位
	GetDevicePoints(id string) (points map[string]DevicePoint, err error)
	// GetDevicePointDetails 获取设备点位详情
	GetDevicePointDetails(id, pointName string) (point DevicePoint, err error)

	// GetDeviceUpdateAt 获取设备最后更新时间
	GetDeviceUpdateAt(id string) (time.Time, error)
	// GetDeviceStatus 获取设备在离线状态
	GetDeviceStatus(id string) (online bool, err error)

	// SetOnline 设置设备为在线状态
	SetOnline(id string) (err error)
	// SetOffline 设置设备为离线状态
	SetOffline(id string) (err error)

	// MayBeOffline 可能离线事件（60秒内超过3次判定离线）
	MayBeOffline(id string) (err error)

	// SetOnlineChangeCallback 设置设备在线状态变化回调函数
	SetOnlineChangeCallback(handlerFunc OnlineChangeCallback)

	// StopStatusListener 停止设备状态监听
	StopStatusListener()

	// GetDevices 获取所有设备
	GetDevices() []Device

	// SetWritePointValue 存储下发控制点位值
	SetWritePointValue(id string, pointName string, value interface{}) (err error)
	// GetWritePointValue 获取下发控制点位值
	GetWritePointValue(id string, pointName string) (value interface{}, err error)
}
