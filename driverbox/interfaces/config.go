package interfaces

import "github.com/ibuilding-x/driver-box/driverbox/dto"

type Config interface {
	GetModels() []dto.Model
	GetDevices() []dto.Device

	GetModelByName(name string) (model dto.Model, ok bool)

	GetPointsByModelName(modelName string) (points []dto.Point, ok bool)
	GetPointsByDeviceId(deviceId string) (points []dto.Point, ok bool)

	GetDeviceById(id string) (dto.Device, error)

	GetProtocolNameByModelName(modelName string) string
	GetProtocolNameByDeviceId(deviceId string) string
}
