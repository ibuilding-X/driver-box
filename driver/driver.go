package driver

import (
	"driver-box/config"
	"driver-box/core/contracts"
	"driver-box/core/helper"
	"driver-box/driver/bootstrap"
	"errors"
	"fmt"
	sdkModels "github.com/edgexfoundry/device-sdk-go/v2/pkg/models"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/models"
	"go.uber.org/zap"
	"sync"
	"time"
)

var driverInstance *Driver
var once = &sync.Once{}

type Driver struct {
	lc            logger.LoggingClient                // EdgeX 日志组件
	serviceConfig *config.ServiceConfig               // 自定义 EdgeX 配置
	deviceCh      chan<- []sdkModels.DiscoveredDevice // 设备通道，暂无使用价值
}

// NewDriver return driver instance
func NewDriver() *Driver {
	once.Do(func() {
		driverInstance = &Driver{}
	})

	return driverInstance
}

// Initialize 初始化
func (s *Driver) Initialize(lc logger.LoggingClient, asyncCh chan<- *sdkModels.AsyncValues, deviceCh chan<- []sdkModels.DiscoveredDevice) error {
	s.lc = lc
	helper.MessageBus = asyncCh
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

		if err := bootstrap.LoadPlugins(); err != nil {
			lc.Errorf("init error: %s", err.Error())
		}

		lc.Info("-------------------- end init --------------------")
	}()

	return nil
}

// HandleReadCommands triggers a protocol Read operation for the specified device.
func (s *Driver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest) (res []*sdkModels.CommandValue, err error) {
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
func (s *Driver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []sdkModels.CommandRequest, params []*sdkModels.CommandValue) error {
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

func (s *Driver) Stop(force bool) error {
	// Then Logging Client might not be initialized
	if s.lc != nil {
		s.lc.Debugf("Driver.Stop called: force=%v", force)
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
func (s *Driver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	helper.Logger.Info("a new Device is added", zap.String("deviceName", deviceName), zap.Any("protocols", protocols))
	return nil
}

// UpdateDevice is a callback function that is invoked
// when a Device associated with this Device Service is updated
func (s *Driver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	helper.Logger.Info("Device is updated", zap.String("deviceName", deviceName), zap.Any("protocols", protocols))
	return nil
}

// RemoveDevice is a callback function that is invoked
// when a Device associated with this Device Service is removed
func (s *Driver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	helper.Logger.Info("Device is removed", zap.String("deviceName", deviceName), zap.Any("protocols", protocols))
	return nil
}
