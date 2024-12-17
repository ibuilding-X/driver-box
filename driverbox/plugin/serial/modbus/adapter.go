package modbus

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/serial"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

var Adapter = &modbusAdapter{}

type primaryTable string

const BatchReadMode plugin.EncodeMode = "batchRead"
const (
	Coil            primaryTable = "COIL"             // 线圈
	DiscreteInput   primaryTable = "DISCRETE_INPUT"   // 离散输入
	InputRegister   primaryTable = "INPUT_REGISTER"   // 离散寄存器
	HoldingRegister primaryTable = "HOLDING_REGISTER" // 保持寄存器
)

// 采集组
type slaveDevice struct {
	// 通讯设备，采集点位可以对应多个物模型设备
	unitID uint8
	//分组
	pointGroup []*pointGroup
}

type pointGroup struct {
	serial.TimerGroup
	// 从机地址
	UnitID uint8
	//寄存器类型
	RegisterType primaryTable
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

// ConnectionConfig 连接器配置
type ConnectionConfig struct {
	Enable        bool   `json:"enable"`        //当前连接是否可用
	Address       string `json:"address"`       // 地址：例如：127.0.0.1:502
	Mode          string `json:"mode"`          // 连接模式：rtuovertcp、rtu
	BaudRate      uint   `json:"baudRate"`      // 波特率（仅串口模式）
	DataBits      uint   `json:"dataBits"`      // 数据位（仅串口模式）
	StopBits      uint   `json:"stopBits"`      // 停止位（仅串口模式）
	Parity        uint   `json:"parity"`        // 奇偶性校验（仅串口模式）
	BatchReadLen  uint16 `json:"batchReadLen"`  // 最长连续读个数
	BatchWriteLen uint16 `json:"batchWriteLen"` // 支持连续写的最大长度
	MinInterval   uint16 `json:"minInterval"`   // 最小读取间隔
	Timeout       uint16 `json:"timeout"`       // 请求超时
	Retry         int    `json:"retry"`         // 重试次数
	Virtual       bool   `json:"virtual"`       //虚拟设备功能
}
type modbusAdapter struct {
	connector *serial.Connector
	devices   map[uint8]*slaveDevice
}

func (adapter *modbusAdapter) InitTimerGroup(connector *serial.Connector) []serial.TimerGroup {
	helper.Logger.Info("init modbus adapter")
	adapter.connector = connector
	groups := make([]serial.TimerGroup, 0)

	connConfig := adapter.connector.Plugin.Config.Connections[connector.ConnectionKey]
	connectionConfig := new(ConnectionConfig)
	if err := helper.Map2Struct(connConfig, connectionConfig); err != nil {
		helper.Logger.Error("convert connector config error", zap.Any("connection", connConfig), zap.Error(err))
		return nil
	}
	//生成点位采集组
	for _, model := range connector.Plugin.Config.DeviceModels {
		for _, dev := range model.Devices {
			if dev.ConnectionKey != adapter.connector.ConnectionKey {
				continue
			}
			adapter.createPointGroup(connectionConfig, model, dev)
		}
	}
	for _, device := range adapter.devices {
		for _, group := range device.pointGroup {
			groups = append(groups, group.TimerGroup)
		}
	}
	return groups
}

// 采集任务分组
func (adapter *modbusAdapter) createPointGroup(conf *ConnectionConfig, model config.DeviceModel, dev config.Device) {
	for _, point := range model.DevicePoints {
		p := point.ToPoint()
		if p.ReadWrite != config.ReadWrite_R && p.ReadWrite != config.ReadWrite_RW {
			continue
		}
		ext, err := convToPointExtend(p.Extends)
		if err != nil {
			helper.Logger.Error("error modbus point config", zap.String("deviceId", dev.ID), zap.Any("point", point), zap.Error(err))
			continue
		}
		ext.Name = p.Name
		ext.DeviceId = dev.ID
		duration, err := time.ParseDuration(ext.Duration)
		if err != nil {
			helper.Logger.Error("error modbus duration config", zap.String("deviceId", dev.ID), zap.Any("config", p.Extends), zap.Error(err))
			duration = time.Second
		}

		device, err := adapter.createDevice(dev.Properties)
		if err != nil {
			helper.Logger.Error("error modbus device config", zap.String("deviceId", dev.ID), zap.Any("config", p.Extends), zap.Error(err))
			continue
		}
		ok := false
		//同一寄存器地址可能通过位运算对应多个点位，也需要将该点加入group
		for _, group := range device.pointGroup {
			//相同采集频率为同一组
			if group.Duration != duration {
				continue
			}
			//不同寄存器类型不为一组
			if group.RegisterType != ext.RegisterType {
				continue
			}

			//如果ext和group中的其他addres区间长度不超过maxLen，则添加到group中
			start := group.Address
			if start > ext.Address {
				start = ext.Address
			}
			end := group.Address + group.Quantity
			if end < ext.Address+ext.Quantity {
				end = ext.Address + ext.Quantity
			}
			//超过最大长度，拆成新的一组
			if end-start <= conf.BatchReadLen {
				group.Points = append(group.Points, ext)
				ok = true
				group.Address = start
				group.Quantity = end - start
				break
			}
		}
		//新增一个点位组
		if !ok {
			ext.DeviceId = dev.ID
			ext.Name = p.Name
			device.pointGroup = append(device.pointGroup, &pointGroup{
				TimerGroup: serial.TimerGroup{
					UUID:     uuid.New().String(),
					Duration: duration,
				},
				UnitID:       device.unitID,
				RegisterType: ext.RegisterType,
				Address:      ext.Address,
				Quantity:     ext.Quantity,
				Points: []*Point{
					ext,
				},
			})
		}
	}

}
func (adapter *modbusAdapter) createDevice(properties map[string]string) (d *slaveDevice, err error) {
	unitID, err := getUnitId(properties)
	d, ok := adapter.devices[unitID]
	if ok {
		return d, nil
	}

	var group []*pointGroup
	d = &slaveDevice{
		unitID:     unitID,
		pointGroup: group,
	}
	adapter.devices[unitID] = d
	return d, nil
}

func getUnitId(properties map[string]string) (uint8, error) {
	unitID := properties["unitID"]
	if len(unitID) == 0 {
		return 0, errors.New("none unitID")
	}
	v, e := strconv.ParseUint(unitID, 10, 8)
	if e != nil {
		return 0, e
	} else {
		return uint8(v), nil
	}
}

func convToPointExtend(extends map[string]interface{}) (*Point, error) {
	extend := new(Point)
	if err := helper.Map2Struct(extends, extend); err != nil {
		helper.Logger.Error("error modbus config", zap.Any("config", extends), zap.Error(err))
		return nil, err
	}
	//未设置，则默认每秒采集一次
	if extend.Duration == "" {
		extend.Duration = "1s"
	}
	//寄存器地址换算
	startAddress, ok := extends["startAddress"]
	if !ok {
		return nil, fmt.Errorf("start address missed")
	}
	address, err := castModbusAddress(startAddress)
	if err != nil {
		return nil, fmt.Errorf("convert start address error: %s", err.Error())
	}
	extend.Address = address

	switch extend.RegisterType {
	case Coil, DiscreteInput: // 线圈及离散输入仅支持读一个
		extend.Quantity = 1
	case InputRegister, HoldingRegister:
		switch strings.ToUpper(extend.RawType) {
		case strings.ToUpper(common.ValueTypeUint16), strings.ToUpper(common.ValueTypeInt16):
			extend.Quantity = 1
		case strings.ToUpper(common.ValueTypeUint32), strings.ToUpper(common.ValueTypeInt32), strings.ToUpper(common.ValueTypeFloat32):
			extend.Quantity = 2
		case strings.ToUpper(common.ValueTypeUint64), strings.ToUpper(common.ValueTypeInt64), strings.ToUpper(common.ValueTypeFloat64):
			extend.Quantity = 4
		case strings.ToUpper(common.ValueTypeString):
			if extend.Quantity == 0 {
				extend.Quantity = 1
			}
		default:
			return nil, fmt.Errorf("unsupported raw type %s", extend.RawType)
		}
	default:
		return nil, fmt.Errorf("unsuported register type: %s", extend.RegisterType)
	}
	return extend, nil
}

func (adapter *modbusAdapter) ExecuteTimerGroup(group *serial.TimerGroup) error {
	helper.Logger.Info("..")

	return nil
	//return &serial.Command{
	//	Mode:        plugin.ReadMode,
	//	MessageType: "modbus",
	//	OutputFrame: "01 03 00 00 00 01 84 0A",
	//}
}
func (adapter *modbusAdapter) DriverBoxEncode(deviceId string, mode plugin.EncodeMode, values ...plugin.PointData) (res []serial.Command, err error) {
	return nil, nil
}
func (adapter *modbusAdapter) DriverBoxDecode(command serial.Command) (res []plugin.DeviceData, err error) {
	return nil, nil
}
func (adapter *modbusAdapter) SendCommand(cmd serial.Command) error {
	return nil
}
func (adapter *modbusAdapter) Release() (err error) {
	return err
}

// castModbusAddress modbus 地址转换
func castModbusAddress(i interface{}) (address uint16, err error) {
	s := cast.ToString(i)
	if strings.HasPrefix(s, "0x") { //check hex format
		addr, err := strconv.ParseInt(s[2:], 16, 32)
		if err != nil {
			return 0, err
		}
		return cast.ToUint16(addr), nil
	} else if strings.HasSuffix(s, "d") {
		addr, err := strconv.Atoi(strings.ReplaceAll(s, "d", ""))
		if err != nil {
			return 0, err
		}
		return cast.ToUint16(addr), nil
	} else if len(s) == 5 { //handle modbus format
		res, err := cast.ToUint16E(s)
		if err != nil {
			return 0, err
		}
		if res > 0 && res < 10000 {
			return res - 1, nil
		} else if res > 10000 && res < 20000 {
			return res - 10001, nil
		} else if res > 30000 && res < 40000 {
			return res - 30001, nil
		} else if res > 40000 && res < 50000 {
			return res - 40001, nil
		} else {
			return 0, fmt.Errorf("invalid modbus address: %v", s)
		}
	}

	res, err := cast.ToUint16E(i)
	if err != nil {
		return 0, err
	}
	return res, nil
}
