package device

import (
	"driver-box/core/helper"
	"fmt"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	models2 "github.com/edgexfoundry/go-mod-core-contracts/v2/models"
)

// Init 初始化
func Init() error {
	return initModelAndDevice()
}

// 初始化模型和设备
func initModelAndDevice() error {
	s := service.RunningService()
	models := helper.CoreCache.Models()
	for _, model := range models {
		// 设备资源
		deviceResources, err := model.Points2Resources()
		if err != nil {
			return err
		}
		// 创建特殊资源 _equip_standard
		specialResource := models2.DeviceResource{
			Description: "设备标准模型",
			Name:        "_equip_standard",
			IsHidden:    false,
			Tag:         "",
			Properties: models2.ResourceProperties{
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
		profile, err := s.GetProfileByName(model.Name)
		if err != nil { // 添加
			profile = models2.DeviceProfile{
				Name:            model.Name,
				Description:     model.Description,
				Manufacturer:    "unknown",
				Model:           "unknown",
				Labels:          []string{"dynamic"},
				DeviceResources: deviceResources,
			}
			_, err = s.AddDeviceProfile(profile)
			if err != nil {
				return err
			}
		} else { // 更新
			profile.Description = model.Description
			profile.DeviceResources = deviceResources
			err = s.UpdateDeviceProfile(profile)
			if err != nil {
				return err
			}
		}

		// 初始化设备
		for _, device := range model.Devices {
			findDevice, err := s.GetDeviceByName(device.Name)
			protocols, ok := helper.CoreCache.GetProtocolsByDevice(device.Name)
			if !ok {
				return fmt.Errorf("device %s protocols not found", device.Name)
			}
			if err != nil { // 添加
				_, err = s.AddDevice(models2.Device{
					Name:           device.Name,
					Description:    device.Description,
					AdminState:     models2.Unlocked,
					OperatingState: models2.Up,
					Protocols:      protocols,
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
				findDevice.Protocols = protocols
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
