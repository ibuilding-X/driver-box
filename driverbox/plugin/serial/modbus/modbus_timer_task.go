package modbus

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/serial"
)

var ModbusTimerTask = &modbusTimerTask{}

type primaryTable string

const BatchReadMode plugin.EncodeMode = "batchRead"
const (
	Coil            primaryTable = "COIL"             // 线圈
	DiscreteInput   primaryTable = "DISCRETE_INPUT"   // 离散输入
	InputRegister   primaryTable = "INPUT_REGISTER"   // 离散寄存器
	HoldingRegister primaryTable = "HOLDING_REGISTER" // 保持寄存器
)

type modbusGroup struct {
	serial.TimerGroup
	// 从机地址
	UnitID uint8
	//起始地址
	Address uint16
	//数量
	Quantity uint16
	Points   []*Point
}
type Point struct {
	config.Point
	//冗余设备相关信息
	DeviceId string

	//点位采集周期
	Duration     string `json:"duration"`
	Address      uint16
	RegisterType primaryTable `json:"primaryTable"`
	//该配置无需设置
	Quantity uint16 `json:"-"`
	Bit      uint8  `json:"bit"`
	BitLen   uint8  `json:"bitLen"`
	RawType  string `json:"rawType"`
	ByteSwap bool   `json:"byteSwap"`
	WordSwap bool   `json:"wordSwap"`
	//写操作是否强制要求多寄存器写接口。某些设备点位虽然只占据一个寄存器地址，但要求采用多寄存器写接口
	MultiWrite bool `json:"multiWrite"`
}
type modbusTimerTask struct {
}

func (task *modbusTimerTask) Init(config config.Config) []serial.TimerGroup {
	helper.Logger.Info("init modbus timer task")
	groups := make([]serial.TimerGroup, 0)
	groups = append(groups, serial.TimerGroup{})
	return groups
}

func (task *modbusTimerTask) EncodeToCommand(group *serial.TimerGroup) *serial.Command {
	helper.Logger.Info("EncodeToCommand..")
	return &serial.Command{
		Mode:        plugin.ReadMode,
		MessageType: "modbus",
		OutputFrame: "01 03 00 00 00 01 84 0A",
	}
}
