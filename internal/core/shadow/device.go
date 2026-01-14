package shadow

import (
	"sync"
	"time"

	"github.com/ibuilding-x/driver-box/driverbox/shadow"
)

// device 设备内部结构
type device struct {
	id              string                         // 设备 ID
	modelName       string                         // 设备模型名称
	points          map[string]*shadow.DevicePoint // 设备点位列表
	online          bool                           // 在线状态
	ttl             time.Duration                  // 设备离线阈值，超过该时长没有收到数据视为离线
	disconnectTimes int                            // 断开连接次数，60秒内超过3次判定离线
	updatedAt       time.Time                      // 更新时间（用于设备离线判断）
	mutex           *sync.RWMutex                  // 锁
}

func newDevice(id string, modelName string, ttl time.Duration) *device {
	return &device{
		id:              id,
		modelName:       modelName,
		points:          make(map[string]*shadow.DevicePoint),
		online:          false,
		ttl:             ttl,
		disconnectTimes: 0,
		updatedAt:       time.Time{},
		mutex:           &sync.RWMutex{},
	}
}

func newDevicePoint(name string) *shadow.DevicePoint {
	return &shadow.DevicePoint{
		Name: name,
	}
}

func (d *device) setPointValue(name string, value interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 更新设备最后更新时间
	updatedAt := time.Now()
	d.updatedAt = updatedAt

	// 重置设备断开连接次数
	d.disconnectTimes = 0

	// 初始化设备点位
	if d.points[name] == nil {
		d.points[name] = newDevicePoint(name)
	}

	// 更新设备点位值
	d.points[name].Value = value
	d.points[name].UpdatedAt = updatedAt
}

func (d *device) getPointValue(name string) (interface{}, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// 1. 离线判断
	if d.online == false {
		return nil, false
	}

	if d.points[name] != nil {
		// 2. 过期判断
		if time.Since(d.points[name].UpdatedAt) <= d.ttl {
			return d.points[name].Value, true
		}
	}

	return nil, false
}

func (d *device) setWritePointValue(name string, value interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 初始化设备点位
	if d.points[name] == nil {
		d.points[name] = newDevicePoint(name)
	}

	// 更新设备点位值
	d.points[name].WriteValue = value
	d.points[name].WriteAt = time.Now()
}

func (d *device) getWritePointValue(name string) (interface{}, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.points[name] != nil {
		return d.points[name].WriteValue, true
	}

	return nil, false
}

func (d *device) getPoint(name string) (*shadow.DevicePoint, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.points[name] != nil {
		return d.points[name], true
	}

	return &shadow.DevicePoint{}, false
}

func (d *device) toPublic() shadow.Device {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// 点位转换
	points := make(map[string]shadow.DevicePoint, len(d.points))
	for pointName, point := range d.points {
		if d.points[pointName] != nil {
			points[pointName] = toPublic(point)
		}
	}

	return shadow.Device{
		ID:              d.id,
		ModelName:       d.modelName,
		Points:          points,
		Online:          d.online,
		TTL:             d.ttl.String(),
		DisconnectTimes: d.disconnectTimes,
		UpdatedAt:       d.updatedAt,
	}
}

func (d *device) getUpdatedAt() time.Time {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.updatedAt
}

func (d *device) getOnline() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.online
}

// setOnline 设置设备在线状态
// 返回值为 true 表示状态变化，false 表示状态未变化
func (d *device) setOnline(online bool) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.online == online {
		return false
	}

	d.online = online
	return true
}

// maybeOffline 设备可能离线
// 返回值为 true 表示本次设定满足离线条件，设备可能离线
func (d *device) maybeOffline() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 已离线，不做处理
	if d.online == false {
		return false
	}

	// 离线阈值判断
	d.disconnectTimes++
	if time.Since(d.updatedAt).Seconds() > 60 && d.disconnectTimes >= 3 {
		d.online = false
		return true
	}

	return false
}

// refreshStatus 刷新设备状态
func (d *device) refreshStatus() (old bool, new bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 离线不处理
	if d.online == false {
		return false, false
	}

	// TTL 判定
	if time.Since(d.updatedAt) > d.ttl {
		d.online = false
		return true, false
	}

	return true, true
}

func toPublic(dp *shadow.DevicePoint) shadow.DevicePoint {
	return shadow.DevicePoint{
		Name:       dp.Name,
		Value:      dp.Value,
		WriteValue: dp.WriteValue,
		UpdatedAt:  dp.UpdatedAt,
		WriteAt:    dp.WriteAt,
	}
}
