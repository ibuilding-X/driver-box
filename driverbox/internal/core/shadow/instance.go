package shadow

import (
	"errors"

	"github.com/ibuilding-x/driver-box/driverbox/internal/export"
	"github.com/ibuilding-x/driver-box/driverbox/internal/logger"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/event"
	"github.com/ibuilding-x/driver-box/driverbox/shadow"
	"go.uber.org/zap"

	"sync"
	"time"
)

var ErrUnknownDevice = errors.New("unknown device")
var DeviceShadow shadow.DeviceShadow // 本地设备影子
type deviceShadow struct {
	devices map[string]*device
	ticker  *crontab.Future
	mutex   *sync.RWMutex
}

func NewDeviceShadow() shadow.DeviceShadow {
	ds := &deviceShadow{
		devices: make(map[string]*device),
		mutex:   &sync.RWMutex{},
	}
	ds.ticker, _ = crontab.Instance().AddFunc("5s", func() {
		ds.checkOffline()
	})
	DeviceShadow = ds
	return ds
}

func (d *deviceShadow) AddDevice(id string, modelName string, ttl ...time.Duration) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 已存在
	if d.devices[id] != nil {
		return
	}

	// ttl
	customTTL := 24 * time.Hour
	if len(ttl) > 0 && ttl[0] > 0 {
		customTTL = ttl[0]
	}

	// 添加
	d.devices[id] = newDevice(id, modelName, customTTL)
}

func (d *deviceShadow) GetDevice(id string) (device shadow.Device, ok bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		return d.devices[id].toPublic(), true
	}

	return
}

func (d *deviceShadow) HasDevice(id string) bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		return true
	}

	return false
}

func (d *deviceShadow) DeleteDevice(id ...string) (err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if len(id) == 0 {
		return nil
	}

	for _, v := range id {
		delete(d.devices, v)
	}

	return nil
}

func (d *deviceShadow) SetDevicePoint(id, pointName string, value interface{}) (err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.devices[id] != nil {
		// 更新点位值
		d.devices[id].setPointValue(pointName, value)
		// 更新设备状态
		if d.devices[id].setOnline(true) {
			d.handlerCallback(id, true)
		}
		return
	}

	return ErrUnknownDevice
}

func (d *deviceShadow) GetDevicePoint(id, pointName string) (value interface{}, err error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		if v, ok := d.devices[id].getPointValue(pointName); ok {
			return v, nil
		}
		return nil, nil
	}

	return nil, ErrUnknownDevice
}

func (d *deviceShadow) GetDevicePoints(id string) (points map[string]shadow.DevicePoint, err error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		return d.devices[id].toPublic().Points, nil
	}

	return nil, ErrUnknownDevice
}

func (d *deviceShadow) GetDevicePointDetails(id, pointName string) (point shadow.DevicePoint, err error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		if p, ok := d.devices[id].getPoint(pointName); ok {
			return toPublic(p), nil
		}
		return shadow.DevicePoint{}, nil
	}

	return shadow.DevicePoint{}, ErrUnknownDevice
}

func (d *deviceShadow) GetDeviceUpdateAt(id string) (time.Time, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		return d.devices[id].getUpdatedAt(), nil
	}

	return time.Time{}, ErrUnknownDevice
}

func (d *deviceShadow) GetDeviceStatus(id string) (online bool, err error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		return d.devices[id].getOnline(), nil
	}

	return false, ErrUnknownDevice
}

func (d *deviceShadow) SetOnline(id string) (err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.devices[id] != nil {
		d.devices[id].setOnline(true)
		d.handlerCallback(id, true)
		return
	}

	return ErrUnknownDevice
}

func (d *deviceShadow) SetOffline(id string) (err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.devices[id] != nil {
		d.devices[id].setOnline(false)
		d.handlerCallback(id, false)
		return
	}

	return ErrUnknownDevice
}

func (d *deviceShadow) MayBeOffline(id string) (err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.devices[id] != nil {
		if d.devices[id].maybeOffline() {
			d.handlerCallback(id, false)
		}
		return
	}

	return ErrUnknownDevice
}

func (d *deviceShadow) StopStatusListener() {
	if d.ticker != nil {
		d.ticker.Disable()
		d.ticker = nil
	}
}

func (d *deviceShadow) GetDevices() []shadow.Device {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	var devices []shadow.Device
	for id, _ := range d.devices {
		if d.devices[id] != nil {
			devices = append(devices, d.devices[id].toPublic())
		}
	}

	return devices
}

func (d *deviceShadow) SetWritePointValue(id string, pointName string, value interface{}) (err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.devices[id] != nil {
		d.devices[id].setWritePointValue(pointName, value)
		return
	}

	return ErrUnknownDevice
}

func (d *deviceShadow) GetWritePointValue(id string, pointName string) (value interface{}, err error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.devices[id] != nil {
		if v, ok := d.devices[id].getWritePointValue(pointName); ok {
			return v, nil
		}
	}

	return nil, ErrUnknownDevice
}

func (d *deviceShadow) handlerCallback(id string, online bool) {
	if online {
		logger.Logger.Info("device online", zap.String("deviceId", id))
	} else {
		logger.Logger.Warn("device offline...", zap.String("deviceId", id))
	}
	//触发设备在离线事件
	export.TriggerEvents(event.EventCodeDeviceStatus, id, online)
}

func (d *deviceShadow) checkOffline() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for id, _ := range d.devices {
		if d.devices[id] != nil {
			if old, now := d.devices[id].refreshStatus(); old && !now {
				d.handlerCallback(id, false)
			}
		}
	}
}
