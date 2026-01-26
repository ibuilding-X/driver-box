package mirror

import (
	"errors"
	"sync"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"go.uber.org/zap"
)

const ProtocolName = "mirror"

var instance *Plugin
var once = &sync.Once{}

type Plugin struct {
	connector *connector
	mutex     *sync.Mutex
	//是否已就绪
	ready bool
}

func EnablePlugin() {
	driverbox.EnablePlugin(ProtocolName, NewPlugin())
}

func NewPlugin() *Plugin {
	once.Do(func() {
		instance = &Plugin{}
		instance.mutex = &sync.Mutex{}
		instance.connector = &connector{
			plugin:     instance,
			mirrors:    make(map[string]map[string]rawDevice),
			rawMapping: make(map[string]map[string][]plugin.DeviceData),
		}
	})
	return instance
}

func (p *Plugin) Initialize(c config.DeviceConfig) {
	//生成镜像设备映射关系
	for _, model := range c.DeviceModels {
		deviceCount := len(model.Devices)
		if deviceCount == 0 {
			_ = driverbox.CoreCache().DeleteModel(model.Name)
			driverbox.Log().Warn("delete model because of none device", zap.Any("model", model))
			continue
		}
		if deviceCount > 1 {
			driverbox.Log().Error("mirror only support one device")
			continue
		}
		err := p.UpdateMirrorMapping(model.Model, model.Devices[0])
		if err != nil {
			driverbox.Log().Error("update mirror mapping failed", zap.Error(err))
		}
	}
	//删除无用的连接
	if len(c.DeviceModels) == 0 {
		_ = driverbox.CoreCache().DeleteConnection(MirrorConnectionKey)
	}
	p.ready = true
}

// UpdateMirrorMapping 更新镜像设备映射关系
func (p *Plugin) UpdateMirrorMapping(model config.Model, device config.Device) error {

	p.mutex.Lock()
	defer func() {
		p.mutex.Unlock()
	}()

	if _, ok := p.connector.mirrors[device.ID]; ok {
		return errors.New("mirror device id must be unique")
	}

	p.connector.mirrors[device.ID] = make(map[string]rawDevice)
	for _, point := range model.DevicePoints {
		//原始设备
		rawD, ok1 := point.FieldValue("rawDevice")
		//原始设备点位
		rawP, ok2 := point.FieldValue("rawPoint")
		if !ok1 || !ok2 {
			return errors.New("mirror point must have rawDevice and rawPoint")
		}

		raw := rawD.(string)

		rawPoint := rawP.(string)
		//创建镜像设备与原始设备的映射关系
		p.connector.mirrors[device.ID][point.Name()] = rawDevice{
			deviceId:  raw,
			pointName: rawPoint,
		}

		//真实设备点位与镜像设备的映射关系
		if _, ok := p.connector.rawMapping[raw]; !ok {
			//初始化设备映射
			p.connector.rawMapping[raw] = make(map[string][]plugin.DeviceData)
		}
		rawPointMapping := p.connector.rawMapping[raw]
		if _, ok := rawPointMapping[rawPoint]; !ok {
			//初始化点位映射
			rawPointMapping[rawPoint] = make([]plugin.DeviceData, 0)
		}
		ok := false
		for _, deviceData := range rawPointMapping[rawPoint] {
			if deviceData.ID != device.ID {
				continue
			}
			deviceData.Values = append(deviceData.Values, plugin.PointData{
				PointName: point.Name(),
			})
			ok = true
			break
		}
		if !ok {
			rawPointMapping[rawPoint] = append(rawPointMapping[rawPoint], plugin.DeviceData{
				ID: device.ID,
				Values: []plugin.PointData{
					{
						PointName: point.Name(),
					},
				},
			})
		}
	}
	return nil
}

func (p *Plugin) Connector(deviceSn string) (connector plugin.Connector, err error) {
	return p.connector, nil
}

// 插件是否已就绪
func (p *Plugin) IsReady() bool {
	return p.ready
}

func (p *Plugin) Destroy() error {
	p.connector.mirrors = make(map[string]map[string]rawDevice)
	p.connector.rawMapping = make(map[string]map[string][]plugin.DeviceData)
	p.ready = false
	return nil
}

// Decode Connector.Decode 迁移至 Plugin.Decode
func (p *Plugin) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return p.connector.Decode(raw)
}
