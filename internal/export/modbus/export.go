package mirror

import (
	"sync"
	"time"

	"github.com/goburrow/serial"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/tbrandon/mbserver"
	"go.uber.org/zap"
)

// Modbus Slave插件
// 在library/common_tpl中预定义各物模型的分布区域、点位偏移量等寄存器模版
// 各模型的第一位寄存器定义该模型支持的设备数量上限，数值要求为8的倍数。
// 后续(设备数量上限/8)各寄存器用比特位表示设备是否存在，
//

var driverInstance *Export
var once = &sync.Once{}

type Export struct {
	ready bool
}

func (export *Export) Init() error {
	export.ready = true

	serv := mbserver.NewServer()
	serv.RegisterFunctionHandler(1, mbserver.ReadHoldingRegisters)
	err := serv.ListenTCP("127.0.0.1:1502")
	if err != nil {
		helper.Logger.Error("failed to listen", zap.Error(err))
	}

	err = serv.ListenRTU(&serial.Config{
		Address:  "/dev/ttyACM0",
		BaudRate: 9600,
		DataBits: 8,
		StopBits: 1,
		Parity:   "N",
		Timeout:  10 * time.Second,
		RS485: serial.RS485Config{
			Enabled:            true,
			DelayRtsBeforeSend: 2 * time.Millisecond,
			DelayRtsAfterSend:  3 * time.Millisecond,
			RtsHighDuringSend:  false,
			RtsHighAfterSend:   false,
			RxDuringTx:         false,
		}})
	if err != nil {
		helper.Logger.Error("failed to listen", zap.Error(err))
	}

	defer serv.Close()
	return nil
}
func (export *Export) Destroy() error {
	export.ready = false
	return nil
}

func NewExport() *Export {
	once.Do(func() {
		driverInstance = &Export{}
	})
	return driverInstance
}

// 点位变化触发场景联动
func (export *Export) ExportTo(deviceData plugin.DeviceData) {
	//设备点位已通过
	//if export.plugin.VirtualConnector != nil && len(deviceData.Events) > 0 {
	//	helper.Logger.Info("export to virtual connector", zap.Any("deviceData", deviceData))
	//	callback.OnReceiveHandler(export.plugin.VirtualConnector, deviceData)
	//}
}

// 继承Export OnEvent接口
func (export *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	switch eventValue {

	}
	return nil
}

func (export *Export) IsReady() bool {
	return export.ready
}
