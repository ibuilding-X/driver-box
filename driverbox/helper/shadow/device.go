package shadow

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"log"
	"sort"
	"sync"
	"time"
)

// Device 设备结构
type Device struct {
	id              string        // 设备id
	modelName       string        // 设备模型名称
	points          *sync.Map     // 设备点位列表
	onlineBindPoint string        // 在线状态绑定点位（支持数据类型：bool、string、int、float）
	online          bool          // 在线状态
	ttl             time.Duration // 设备离线阈值，超过该时长没有收到数据视为离线
	disconnectTimes int           // 断开连接次数，60秒内超过3次判定离线
	updatedAt       time.Time     // 更新时间（用于设备离线判断）
}

// DeviceAPI 对外开放设备数据
type DeviceAPI struct {
	ID              string           `json:"id"`
	Points          []DevicePointAPI `json:"points"`
	Online          bool             `json:"online"`
	TTL             string           `json:"ttl"`
	DisconnectTimes int              `json:"disconnect_times"`
	UpdatedAt       string           `json:"updated_at"`
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
	//该点位最近一次执行写操作的时间
	LatestWriteTime time.Time
}

// DevicePointAPI 对外开放设备点位
type DevicePointAPI struct {
	Name      string `json:"name"`
	Value     any    `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

func NewDevice(device config.Device, modelName string, points map[string]DevicePoint) Device {
	//默认24小时无数据上报，视为设备离线
	ttl := time.Duration(24) * time.Hour
	if device.Ttl != "" {
		t, err := time.ParseDuration(device.Ttl)
		if err != nil {
			log.Fatalf("device:%v parse ttl:%v error:%v", device.ID, device.Ttl, err)
		} else {
			ttl = t
		}
	} else {
		log.Printf("device:%v ttl unset, reset default value:%v", device.ID, ttl)
	}
	// 转换 points
	ps := &sync.Map{}
	for k, _ := range points {
		ps.Store(k, points[k])
	}
	return Device{
		id:              device.ID,
		modelName:       modelName,
		points:          ps,
		onlineBindPoint: "",
		ttl:             ttl,
		online:          false, // 默认设备处于离线状态
	}
}

// ToDeviceAPI 转换设备 API
func (d *Device) ToDeviceAPI() DeviceAPI {
	device := DeviceAPI{
		ID:              d.id,
		Points:          make([]DevicePointAPI, 0),
		Online:          d.online,
		TTL:             d.ttl.String(),
		DisconnectTimes: d.disconnectTimes,
		UpdatedAt:       d.updatedAt.Format("2006-01-02 15:04:05"),
	}
	// 重组点位
	d.points.Range(func(_, value any) bool {
		if point, ok := value.(DevicePoint); ok {
			device.Points = append(device.Points, point.ToDevicePointAPI())
		}
		return true
	})

	//按点位名排序
	sort.Slice(device.Points, func(i, j int) bool {
		return device.Points[i].Name < device.Points[j].Name
	})
	return device
}

func (dp DevicePoint) ToDevicePointAPI() DevicePointAPI {
	return DevicePointAPI{
		Name:      dp.Name,
		Value:     dp.Value,
		UpdatedAt: dp.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// GetDevicePoint 获取设备指定点位数据
func (d *Device) GetDevicePoint(pointName string) (DevicePoint, bool) {
	if v, ok := d.points.Load(pointName); ok {
		point, _ := v.(DevicePoint)
		return point, true
	}
	return DevicePoint{}, false
}

// GetDevicePointAPI 获取设备指定点位数据（开放 API 使用）
func (d *Device) GetDevicePointAPI(pointName string) (DevicePointAPI, bool) {
	if point, ok := d.GetDevicePoint(pointName); ok {
		return point.ToDevicePointAPI(), true
	}
	return DevicePointAPI{}, false
}
