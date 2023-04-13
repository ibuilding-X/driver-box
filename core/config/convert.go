// 核心配置转换

package config

import (
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
)

// Points2Resources points 转 resources
// 配置文件点位值类型限定三种：int、float、string，对应 go 类型：int64、float64、string
func (m Model) Points2Resources() (deviceResources []models.DeviceResource, err error) {
	for _, point := range m.Points {
		// 点位类型适配（int64、float64、string）
		valueType := point.ValueType
		switch point.ValueType {
		case "int":
			valueType = common.ValueTypeInt64
		case "float":
			valueType = common.ValueTypeFloat64
		case "string":
			valueType = common.ValueTypeString
		}
		attributes := make(map[string]interface{})
		attributes["realReport"] = point.RealReport
		attributes["reportMode"] = point.ReportMode
		attributes["timerReport"] = point.TimerReport
		if point.Units != "" {
			attributes["units"] = point.Units
		}
		// 默认配置
		deviceResources = append(deviceResources, models.DeviceResource{
			Description: point.Description,
			Name:        point.Name,
			IsHidden:    false,
			Tag:         "",
			Properties: models.ResourceProperties{
				ValueType:    valueType,
				ReadWrite:    point.ReadWrite,
				DefaultValue: point.DefaultValue,
			},
			Attributes: attributes,
		})
	}
	return
}

// Actions2Commands action 转 command
func (dm DeviceModel) Actions2Commands() (deviceCommands []models.DeviceCommand) {
	for _, action := range dm.DeviceActions {
		resourceOperations := make([]models.ResourceOperation, 0)
		for _, operation := range action.ResourceOperations {
			resourceOperations = append(resourceOperations, models.ResourceOperation{
				DeviceResource: operation.DeviceResource,
				DefaultValue:   operation.DefaultValue,
				Mappings:       operation.Mappings,
			})
		}
		deviceCommands = append(deviceCommands, models.DeviceCommand{
			Name:               action.Name,
			IsHidden:           false,
			ReadWrite:          action.ReadWrite,
			ResourceOperations: resourceOperations,
		})
	}
	return
}

//// ConvAutoEvents 转换自动事件
//func (d Device) ConvAutoEvents() (autoEvents []models.AutoEvent) {
//	for _, event := range d.AutoEvents {
//		autoEvents = append(autoEvents, models.AutoEvent{
//			Interval:   event.Interval,
//			SourceName: event.PointName,
//		})
//	}
//	return
//}

func (d Device) ConvProtocols() map[string]models.ProtocolProperties {
	m := make(map[string]models.ProtocolProperties)
	m["other"] = d.Protocol
	return m
}
