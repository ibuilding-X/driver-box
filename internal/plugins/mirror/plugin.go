package mirror

import (
	"errors"
	"sync"

	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

const ProtocolName = "mirror"

var instance *Plugin
var once = &sync.Once{}

type Plugin struct {
	ls        *lua.LState // lua 虚拟机
	connector *connector
	mutex     *sync.Mutex
	//是否已就绪
	ready bool
}

func NewPlugin() *Plugin {
	once.Do(func() {
		instance = &Plugin{}
		instance.mutex = &sync.Mutex{}
		instance.connector = &connector{
			plugin:     instance,
			mirrors:    make(map[string]map[string]Device),
			rawMapping: make(map[string]map[string][]plugin.DeviceData),
		}
	})
	return instance
}

func (p *Plugin) Initialize(logger *zap.Logger, c config.Config, ls *lua.LState) {
	p.ls = ls
	//生成镜像设备映射关系
	for _, model := range c.DeviceModels {
		err := p.UpdateMirrorMapping(model)
		if err != nil {
			logger.Error("update mirror mapping failed", zap.Error(err))
		}
	}
	p.ready = true
}

// UpdateMirrorMapping 更新镜像设备映射关系
func (p *Plugin) UpdateMirrorMapping(model config.DeviceModel) error {
	deviceCount := len(model.Devices)
	if deviceCount == 0 {
		return nil
	}
	if deviceCount > 1 {
		return errors.New("mirror only support one device")
	}
	p.mutex.Lock()
	defer func() {
		p.mutex.Unlock()
	}()

	device := model.Devices[0]
	if _, ok := p.connector.mirrors[device.ID]; ok {
		return errors.New("mirror device id must be unique")
	}

	p.connector.mirrors[device.ID] = make(map[string]Device)
	for _, point := range model.DevicePoints {
		if point.FieldValue("rawDevice") == nil || point.FieldValue("rawPoint") == nil {
			return errors.New("mirror point must have rawDevice and rawPoint")
		}
		//原始设备
		rawDevice := point.FieldValue("rawDevice").(string)
		//原始设备点位
		rawPoint := point.FieldValue("rawPoint").(string)
		//创建镜像设备与原始设备的映射关系
		p.connector.mirrors[device.ID][point.Name()] = Device{
			deviceId:  rawDevice,
			pointName: rawPoint,
		}

		//真实设备点位与镜像设备的映射关系
		if _, ok := p.connector.rawMapping[rawDevice]; !ok {
			//初始化设备映射
			p.connector.rawMapping[rawDevice] = make(map[string][]plugin.DeviceData)
		}
		rawPointMapping := p.connector.rawMapping[rawDevice]
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
	if p.ls != nil {
		helper.Close(p.ls)
	}
	p.connector.mirrors = make(map[string]map[string]Device)
	p.connector.rawMapping = make(map[string]map[string][]plugin.DeviceData)
	p.ready = false
	return nil
}

// Decode Connector.Decode 迁移至 Plugin.Decode
func (p *Plugin) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return p.connector.Decode(raw)
}
