package edgex

import (
	"context"
	"driver-box/config"
	"driver-box/core/contracts"
	"driver-box/core/helper"
	"encoding/json"
	"errors"
	"fmt"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/startup"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/http"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos/requests"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	models2 "github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"go.uber.org/zap"
	"sync"
	"time"
)

var driverInstance *EdgexExport
var once = &sync.Once{}

const (
	serviceName string = "driver-box"
	version     string = "0.0.2"
)

var noticeClient interfaces.NotificationClient
var noticeLabels = []string{"Default"} // 通知固定 Labels
var noticeCategory = "normal"          // 通知固定 Category

type EventType string
type DeviceEventEnum string

const (
	CommandAck      EventType = "CommandAck"      // 指令相应
	ScheduleTrigger EventType = "ScheduleTrigger" // 时间表触发
	DeviceEvent     EventType = "DeviceEvent"     // 设别事件
)

const (
	DeviceEventOnline  DeviceEventEnum = "Online"  // 设备在线事件
	DeviceEventOffline DeviceEventEnum = "Offline" // 设备离线事件
)

// EventModel 自定义事件模型
type EventModel struct {
	EventType       EventType   `json:"eventType"`       // 事件类型（指令相应、时间表触发、设备事件）
	ReportTimestamp int64       `json:"reportTimestamp"` // 上报时间戳（毫秒级）
	EventData       interface{} `json:"eventData"`       // 事件数据
	Description     string      `json:"description"`     // 事件描述
}

// DeviceEventModel 设备事件模型
type DeviceEventModel struct {
	DeviceSN string                 `json:"deviceSN"` // 设备SN
	Type     DeviceEventEnum        `json:"type"`     // 事件名称
	Data     map[string]interface{} `json:"data"`     // 事件数据
}

type EdgexExport struct {
	lc            logger.LoggingClient                // EdgeX 日志组件
	serviceConfig *config.ServiceConfig               // 自定义 EdgeX 配置
	deviceCh      chan<- []sdkModels.DiscoveredDevice // 设备通道，暂无使用价值
	asyncCh       chan<- *sdkModels.AsyncValues
}

func (export *EdgexExport) Init() error {
	startup.Bootstrap(serviceName, version, export)
	return nil
}

// 导出消息：写入Edgex总线、MQTT上云
func (export *EdgexExport) ExportTo(deviceData contracts.DeviceData) {
	// WriteToMessageBus 设备数据写入消息总线
	var values []*sdkModels.CommandValue
	for _, point := range deviceData.Values {
		// 获取点位信息
		cachePoint, ok := helper.CoreCache.GetPointByDevice(deviceData.DeviceName, point.PointName)
		if !ok {
			helper.Logger.Warn("unknown point", zap.Any("deviceName", deviceData.DeviceName), zap.Any("pointName", point.PointName))
			continue
		}
		// 缓存比较
		shadowValue, _ := helper.DeviceShadow.GetDevicePoint(deviceData.DeviceName, point.PointName)
		if shadowValue == point.Value {
			helper.Logger.Debug("point value = cache, stop sending to messageBus")
			continue
		}
		// 缓存
		if err := helper.DeviceShadow.SetDevicePoint(deviceData.DeviceName, point.PointName, point.Value); err != nil {
			helper.Logger.Error("shadow store point value error", zap.Error(err), zap.Any("deviceName", deviceData.DeviceName))
		}
		// 点位类型转换
		pointValue, err := helper.ConvPointType(point.Value, cachePoint.ValueType)
		if err != nil {
			helper.Logger.Warn("point value type convert error", zap.Error(err))
			continue
		}
		// 点位值类型名称转换
		pointType := helper.PointValueType2EdgeX(cachePoint.ValueType)
		v, err := sdkModels.NewCommandValue(point.PointName, pointType, pointValue)
		if err != nil {
			helper.Logger.Warn("new command value error", zap.Error(err), zap.Any("pointName", point.PointName), zap.Any("type", pointType), zap.Any("value", pointValue))
			continue
		}
		values = append(values, v)
	}
	if len(values) > 0 {
		helper.Logger.Info("send to message bus", zap.Any("deviceName", deviceData.DeviceName), zap.Any("values", values))
		export.asyncCh <- &sdkModels.AsyncValues{
			DeviceName:    deviceData.DeviceName,
			SourceName:    "default",
			CommandValues: values,
		}
	}
}

// NewEdgexExport return driver instance
func NewEdgexExport() *EdgexExport {
	once.Do(func() {
		driverInstance = &EdgexExport{}
	})

	return driverInstance
}

func (export *EdgexExport) IsReady() bool {
	return false
}

// Initialize 初始化
func (s *EdgexExport) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModels.AsyncValues, deviceCh chan<- []sdkModels.DiscoveredDevice) error {
	s.lc = lc
	s.asyncCh = asyncCh
	s.deviceCh = deviceCh
	s.serviceConfig = &config.ServiceConfig{}

	ds := service.RunningService()

	if err := ds.LoadCustomConfig(s.serviceConfig, "DriverConfig"); err != nil {
		return fmt.Errorf("unable to load 'DriverConfig' custom configuration: %s", err.Error())
	}

	lc.Infof("DriverConfig config is: %v", s.serviceConfig.DriverConfig)
	helper.DriverConfig = s.serviceConfig.DriverConfig

	if err := s.serviceConfig.DriverConfig.Validate(); err != nil {
		return fmt.Errorf("'DriverConfig' custom configuration validation failed: %s", err.Error())
	}

	// 异步延迟初始化
	// 1. 防止服务未注册
	// 2. 初始化错误不影响服务正常运行
	go func() {
		time.Sleep(time.Second)
		lc.Info("-------------------- begin init --------------------")

		if err := s.initialize(); err != nil {
			lc.Errorf("init error: %s", err.Error())
		}

		// 初始化设备模型及设备
		if err := initModelAndDevice(); err != nil {
			lc.Errorf("device init error: %s", err.Error())
		}
		lc.Info("-------------------- end init --------------------")
	}()

	return nil
}

// initialize 额外初始化工作
func (s *EdgexExport) initialize() error {

	// 初始化通知服务
	url := fmt.Sprintf("http://%s:%d", "edgex-support-notifications", 59860)
	noticeClient = http.NewNotificationClient(url)

	return nil
}

// SendStatusChangeNotification 发送设备状态变更通知
func (s *EdgexExport) SendStatusChangeNotification(deviceName string, online bool) error {
	var reqs []requests.AddNotificationRequest
	noticeData := newStatusChangeNoticeData(deviceName, online)
	notification := dtos.NewNotification(noticeLabels, noticeCategory, noticeData, service.RunningService().ServiceName, models.Normal)
	req := requests.NewAddNotificationRequest(notification)
	reqs = append(reqs, req)

	_, err := noticeClient.SendNotification(context.Background(), reqs)
	return err
}

// newStatusChangeNoticeData 实例化状态变更通知数据
func newStatusChangeNoticeData(deviceName string, online bool) string {
	var status DeviceEventEnum
	if online {
		status = DeviceEventOnline
	} else {
		status = DeviceEventOffline
	}
	data := EventModel{
		EventType:       DeviceEvent,
		ReportTimestamp: time.Now().UnixMilli(),
		EventData: DeviceEventModel{
			DeviceSN: deviceName,
			Type:     status,
		},
	}
	b, _ := json.Marshal(data)
	return string(b)
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (s *EdgexExport) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest) (res []*sdkModels.CommandValue, err error) {
	helper.Logger.Debug("handle read commands", zap.Any("deviceName", deviceName), zap.Any("protocols", protocols), zap.Any("reqs", reqs))

	for i, _ := range reqs {
		if _, ok := helper.CoreCache.GetPointByDevice(deviceName, reqs[i].DeviceResourceName); ok {
			err = helper.Send(deviceName, contracts.ReadMode, contracts.PointData{
				PointName: reqs[i].DeviceResourceName,
				Type:      reqs[i].Type,
			})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New(fmt.Sprintf("device %s not found", deviceName))
		}
	}

	return
}

// HandleWriteCommands passes a slice of CommandRequest struct each representing
// a ResourceOperation for a specific device resource.
// Since the commands are actuation commands, params provide parameters for the individual
// command.
func (s *EdgexExport) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest, params []*sdkModels.CommandValue) error {
	helper.Logger.Debug("handle write commands", zap.Any("deviceName", deviceName), zap.Any("protocols", protocols), zap.Any("reqs", reqs), zap.Any("params", params))

	// 命令参数校验
	if len(reqs) != len(params) {
		msg := "write commands len(reqs) != len(params)"
		return errors.New(msg)
	}
	for i, _ := range reqs {
		if _, ok := helper.CoreCache.GetPointByDevice(deviceName, reqs[i].DeviceResourceName); ok {
			err := helper.Send(deviceName, contracts.WriteMode, contracts.PointData{
				PointName: reqs[i].DeviceResourceName,
				Type:      reqs[i].Type,
				Value:     params[i].Value,
			})
			if err != nil {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("device %s not found", deviceName))
		}
	}
	return nil
}

func (s *EdgexExport) Stop(force bool) error {
	// Then Logging Client might not be initialized
	if s.lc != nil {
		s.lc.Debugf("Stop called: force=%v", force)
	}
	// 释放驱动资源
	if helper.RunningPlugin != nil {
		err := helper.RunningPlugin.Destroy()
		if err != nil {
			return err
		}
		helper.RunningPlugin = nil
	}
	return nil
}

// AddDevice is a callback function that is invoked
// when a new Device associated with this Device Service is added
func (s *EdgexExport) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	helper.Logger.Info("a new Device is added", zap.String("deviceName", deviceName), zap.Any("protocols", protocols))
	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (s *EdgexExport) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	helper.Logger.Info("Device is updated", zap.String("deviceName", deviceName), zap.Any("protocols", protocols))
	return nil
}

// RemoveDevice is a callback function that is invoked
// when a Device associated with this Device Service is removed
func (s *EdgexExport) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	helper.Logger.Info("Device is removed", zap.String("deviceName", deviceName), zap.Any("protocols", protocols))
	return nil
}

// 初始化模型和设备
func initModelAndDevice() error {
	s := service.RunningService()
	models := helper.CoreCache.Models()
	for _, model := range models {
		// 设备资源
		deviceResources, err := Points2Resources(model)
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
			protocols := map[string]models2.ProtocolProperties{
				"fill": {
					"fill": "fill",
				},
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
