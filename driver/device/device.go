package device

import (
	"driver-box/core/config"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
)

// Init 初始化
func Init(c config.Config) error {
	return initModelAndDevice(c)
}

// 初始化模型和设备
func initModelAndDevice(c config.Config) error {
	s := service.RunningService()
	for _, model := range c.DeviceModels {
		// 设备资源
		deviceResources, err := model.Points2Resources()
		if err != nil {
			return err
		}
		// 创建特殊资源 _equip_standard
		specialResource := models.DeviceResource{
			Description: "设备标准模型",
			Name:        "_equip_standard",
			IsHidden:    false,
			Tag:         "",
			Properties: models.ResourceProperties{
				ValueType:    common.ValueTypeString,
				ReadWrite:    common.ReadWrite_R,
				DefaultValue: "",
			},
			Attributes: map[string]interface{}{
				"deviceType":   model.ModelID,
				"deviceName":   "",
				"deviceNameEn": "",
			},
		}
		deviceResources = append(deviceResources, specialResource)
		// 设备命令
		deviceCommands := model.Actions2Commands()
		profile, err := s.GetProfileByName(model.Name)
		if err != nil { // 添加
			profile = models.DeviceProfile{
				Name:            model.Name,
				Description:     model.Description,
				Manufacturer:    "unknown",
				Model:           "unknown",
				Labels:          []string{"dynamic"},
				DeviceResources: deviceResources,
				DeviceCommands:  deviceCommands,
			}
			_, err = s.AddDeviceProfile(profile)
			if err != nil {
				return err
			}
		} else { // 更新
			profile.Description = model.Description
			profile.DeviceResources = deviceResources
			profile.DeviceCommands = deviceCommands
			err = s.UpdateDeviceProfile(profile)
			if err != nil {
				return err
			}
		}

		// 初始化设备
		for _, device := range model.Devices {
			findDevice, err := s.GetDeviceByName(device.Name)
			if err != nil { // 添加
				_, err = s.AddDevice(models.Device{
					Name:           device.Name,
					Description:    device.Description,
					AdminState:     models.Unlocked,
					OperatingState: models.Up,
					Protocols:      device.ConvProtocols(),
					Labels:         []string{"dynamic"},
					Location:       nil,
					ServiceName:    s.ServiceName,
					ProfileName:    model.Name,
					//AutoEvents:     device.ConvAutoEvents(),
					AutoEvents: nil,
					Notify:     false,
				})
				if err != nil {
					return err
				}
			} else { // 更新
				findDevice.Description = device.Description
				findDevice.Protocols = device.ConvProtocols()
				findDevice.ServiceName = s.ServiceName
				findDevice.ProfileName = model.Name
				if err = s.UpdateDevice(findDevice); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
