package modbusserver

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/mbserver"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/mbserver/modbus"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
)

type Export struct {
	server mbserver.Server
}

func (e *Export) Init() error {
	conf := &mbserver.ServerConfig{
		Config:  modbus.DefaultSerialConfig(),
		Models:  e.getConvertedModels(),
		Devices: e.getConvertedDevices(),
	}

	ser := mbserver.NewServer(conf)
	if err := ser.Start(); err != nil {
		helper.Logger.Error("start modbus server error", zap.Error(err))
	}

	return nil
}

func (e *Export) ExportTo(deviceData plugin.DeviceData) {
	//TODO implement me
	panic("implement me")
}

func (e *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	return nil
}

func (e *Export) IsReady() bool {
	//TODO implement me
	panic("implement me")
}

func (e *Export) convertValueType(valueType config.ValueType) (int, error) {
	switch valueType {
	case config.ValueType_Int:
		return mbserver.ValueTypeUint16, nil
	case config.ValueType_Float:
		return mbserver.ValueTypeFloat64, nil
	default:
		return 0, fmt.Errorf("unsupported value type: %v", valueType)
	}
}

func (e *Export) convertReadWrite(rw config.ReadWrite) (int, error) {
	switch rw {
	case config.ReadWrite_R:
		return mbserver.AccessRead, nil
	case config.ReadWrite_RW:
		return mbserver.AccessReadWrite, nil
	case config.ReadWrite_W:
		return mbserver.AccessWrite, nil
	default:
		return 0, fmt.Errorf("unsupported read write: %v", rw)
	}
}

func (e *Export) convertPoints(points map[string]config.Point) []mbserver.Property {
	var result []mbserver.Property

	for _, point := range points {
		valueType, err := e.convertValueType(point.ValueType)
		if err != nil {
			continue
		}

		access, err := e.convertReadWrite(point.ReadWrite)
		if err != nil {
			continue
		}

		result = append(result, mbserver.Property{
			Name:        point.Name,
			Description: point.Description,
			ValueType:   valueType,
			Access:      access,
		})
	}

	return result
}

func (e *Export) getConvertedModels() []mbserver.Model {
	var result []mbserver.Model

	models := helper.CoreCache.Models()
	for _, model := range models {
		result = append(result, mbserver.Model{
			Id:         model.Name,
			Name:       model.Description,
			Properties: e.convertPoints(model.Points),
		})
	}

	return result
}

func (e *Export) getConvertedDevices() []mbserver.Device {
	var result []mbserver.Device

	devices := helper.CoreCache.Devices()
	for _, device := range devices {
		result = append(result, mbserver.Device{
			ModelId: device.ModelName,
			Id:      device.ID,
		})
	}

	return result
}
