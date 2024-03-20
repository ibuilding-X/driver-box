package shadow

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var UnknownDeviceErr = errors.New("unknown device")
var DeviceRepeatErr = errors.New("device already exists")
var BindPointDataErr = errors.New("bind online point data can't be parsed")
var UnknownDevicePointErr = errors.New("unknown device point")

type OnlineChangeCallback func(deviceSn string, online bool) // 设备上/下线回调

// DeviceShadow 设备影子
type DeviceShadow interface {
	AddDevice(device Device) (err error)
	GetDevice(deviceSn string) (device Device, err error)

	SetDevicePoint(deviceSn, pointName string, value interface{}) (err error)
	GetDevicePoint(deviceSn, pointName string) (value interface{}, err error)
	GetDevicePoints(deviceSn string) (points map[string]DevicePoint, err error)

	GetDeviceUpdateAt(deviceSn string) (time.Time, error)

	GetDeviceStatus(deviceSn string) (online bool, err error)

	SetOnline(deviceSn string) (err error)
	SetOffline(deviceSn string) (err error)

	// MayBeOffline 可能离线事件（60秒内超过3次判定离线）
	MayBeOffline(deviceSn string) (err error)

	SetOnlineChangeCallback(handlerFunc OnlineChangeCallback)

	// StopStatusListener 停止设备状态监听
	StopStatusListener()

	// GetDevices 获取所有设备
	GetDevices() []Device
}

type deviceShadow struct {
	m           *sync.Map
	ticker      *time.Ticker
	handlerFunc OnlineChangeCallback
}

func NewDeviceShadow() DeviceShadow {
	shadow := &deviceShadow{
		m:      &sync.Map{},
		ticker: time.NewTicker(time.Second),
	}
	go shadow.checkOnOff()
	return shadow
}

func (d *deviceShadow) AddDevice(device Device) (err error) {
	if _, ok := d.m.Load(device.deviceSn); ok {
		return DeviceRepeatErr
	}
	device.updatedAt = time.Now()
	d.m.Store(device.deviceSn, device)
	return nil
}

func (d *deviceShadow) GetDevice(deviceSn string) (device Device, err error) {
	if deviceAny, ok := d.m.Load(deviceSn); ok {
		return deviceAny.(Device), nil
	} else {
		return Device{}, UnknownDeviceErr
	}
}

func (d *deviceShadow) SetDevicePoint(deviceSn, pointName string, value interface{}) (err error) {
	deviceAny, ok := d.m.Load(deviceSn)
	if !ok {
		return UnknownDeviceErr
	}
	device, _ := deviceAny.(Device)
	if device.points == nil {
		device.points = &sync.Map{}
	}
	// update point value
	device.updatedAt = time.Now()
	device.disconnectTimes = 0
	device.points.Store(pointName, DevicePoint{
		Name:      pointName,
		Value:     value,
		UpdatedAt: time.Now(),
	})
	// update device online status
	if device.onlineBindPoint == pointName { // bind point
		if online, err := parseOnlineBindPV(value); err == nil {
			if device.online != online {
				device.online = online
				d.handlerCallback(deviceSn, online)
			}
		}
	} else { // not bind point
		if device.online != true {
			device.online = true
			d.handlerFunc(deviceSn, true)
		}
	}
	// update
	d.m.Store(deviceSn, device)
	return
}

func (d *deviceShadow) GetDevicePoint(deviceSn, pointName string) (value interface{}, err error) {
	if deviceAny, ok := d.m.Load(deviceSn); ok {
		device, _ := deviceAny.(Device)
		// 1. 设备离线
		if device.online == false {
			return
		}
		// 2. 点位缓存过期
		if pointAny, exist := device.points.Load(pointName); exist {
			point, _ := pointAny.(DevicePoint)
			if time.Since(point.UpdatedAt) > device.ttl {
				return
			}
			return point.Value, nil
		}
		return nil, UnknownDevicePointErr
	} else {
		return nil, UnknownDeviceErr
	}
}

func (d *deviceShadow) GetDevicePoints(deviceSn string) (points map[string]DevicePoint, err error) {
	if deviceAny, ok := d.m.Load(deviceSn); ok {
		ps := make(map[string]DevicePoint)
		deviceAny.(Device).points.Range(func(key, value any) bool {
			k, _ := key.(string)
			v, _ := value.(DevicePoint)
			ps[k] = v
			return true
		})
		return ps, nil
	} else {
		return nil, UnknownDeviceErr
	}
}

func (d *deviceShadow) GetDeviceUpdateAt(deviceSn string) (time.Time, error) {
	if deviceAny, ok := d.m.Load(deviceSn); ok {
		return deviceAny.(Device).updatedAt, nil
	} else {
		return time.Time{}, UnknownDeviceErr
	}
}

func (d *deviceShadow) changeOnOff(deviceSn string, online bool) (err error) {
	if deviceAny, ok := d.m.Load(deviceSn); ok {
		device := deviceAny.(Device)
		if device.online != online {
			device.online = online
			device.updatedAt = time.Now()
			device.disconnectTimes = 0
			d.m.Store(deviceSn, device)
			d.handlerCallback(deviceSn, online)
		}
	} else {
		return UnknownDeviceErr
	}
	return
}

func (d *deviceShadow) GetDeviceStatus(deviceSn string) (online bool, err error) {
	if deviceAny, ok := d.m.Load(deviceSn); ok {
		device := deviceAny.(Device)
		return device.online, nil
	} else {
		return false, UnknownDeviceErr
	}
}

func (d *deviceShadow) SetOnline(deviceSn string) (err error) {
	return d.changeOnOff(deviceSn, true)
}

func (d *deviceShadow) SetOffline(deviceSn string) (err error) {
	return d.changeOnOff(deviceSn, false)
}

func (d *deviceShadow) SetOnlineChangeCallback(handlerFunc OnlineChangeCallback) {
	d.handlerFunc = handlerFunc
}

func (d *deviceShadow) MayBeOffline(deviceSn string) (err error) {
	if deviceAny, ok := d.m.Load(deviceSn); ok {
		device := deviceAny.(Device)
		if device.online == false {
			return
		}
		device.disconnectTimes++
		if time.Since(device.updatedAt).Seconds() > 60 && device.disconnectTimes >= 3 {
			return d.SetOffline(deviceSn)
		}
		// 更新设备信息
		d.m.Store(deviceSn, device)
		return
	} else {
		return UnknownDeviceErr
	}
}

func (d *deviceShadow) StopStatusListener() {
	d.ticker.Stop()
}

// GetDevices 获取所有设备
func (d *deviceShadow) GetDevices() []Device {
	list := make([]Device, 0)
	d.m.Range(func(_, value any) bool {
		if device, ok := value.(Device); ok {
			list = append(list, device)
		}
		return true
	})
	return list
}

func (d *deviceShadow) checkOnOff() {
	for range d.ticker.C {
		d.m.Range(func(key, value any) bool {
			if device, ok := value.(Device); ok {
				if device.online && time.Since(device.updatedAt) > device.ttl {
					_ = d.SetOffline(device.deviceSn)
				}
			}
			return true
		})
	}
}

func (d *deviceShadow) handlerCallback(deviceSn string, online bool) {
	if d.handlerFunc != nil {
		go d.handlerFunc(deviceSn, online)
	}
}

// 解析在离线状态绑定点位值（支持数据类型：int、float、string、bool）
func parseOnlineBindPV(pv interface{}) (online bool, err error) {
	switch v := pv.(type) {
	case string:
		return parseStrOnlineBindPV(v)
	case int8, int16, int32, int, int64, uint8, uint16, uint32, uint, uint64:
		s := fmt.Sprintf("%d", v)
		return parseStrOnlineBindPV(s)
	case float32, float64:
		s := fmt.Sprintf("%.0f", v)
		return parseStrOnlineBindPV(s)
	case bool:
		if v {
			return true, nil
		}
		return
	default:
		return false, BindPointDataErr
	}
}

func parseStrOnlineBindPV(pv string) (online bool, err error) {
	onlineList := []string{"on", "online", "1", "true", "yes"}
	offlineList := []string{"off", "offline", "0", "false", "no"}
	for i, _ := range onlineList {
		if pv == onlineList[i] {
			return true, nil
		}
	}
	for i, _ := range offlineList {
		if pv == offlineList[i] {
			return false, nil
		}
	}
	return false, BindPointDataErr
}
