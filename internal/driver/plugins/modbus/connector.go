package modbus

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/core/contracts"
	"github.com/ibuilding-x/driver-box/core/helper"
	"github.com/ibuilding-x/driver-box/internal/driver/common"
	"github.com/simonvetter/modbus"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
	"time"
)

// connectorConfig 连接器配置
type connectorConfig struct {
	Address  string `json:"address"`   // 地址：例如：127.0.0.1:502
	Mode     string `json:"mode"`      // 连接模式：rtuovertcp、rtu
	SlaveID  uint8  `json:"slave_id"`  // 从机ID
	BaudRate uint   `json:"baud_rate"` // 波特率（仅串口模式）
	DataBits uint   `json:"data_bits"` // 数据位（仅串口模式）
	StopBits uint   `json:"stop_bits"` // 停止位（仅串口模式）
	Parity   uint   `json:"parity"`    // 奇偶性校验（仅串口模式）
	Timeout  int    `json:"timeout"`   // 请求超时
}

// connector 连接器
type connector struct {
	plugin *Plugin
	config connectorConfig
	client *modbus.ModbusClient
	mutex  sync.Mutex
}

type rawData struct {
	DeviceName     string          `json:"deviceName"`
	PointRawValues []pointRawValue `json:"points"`
}

type pointRawValue struct {
	Name      string      `json:"name"`      // 点位名称
	RawValues []uint16    `json:"rawValues"` // 原始数据  按照寄存器顺序排列
	Value     interface{} `json:"value"`     // 按照标准类型解析后的数据
}

// Send 发送数据
func (c *connector) Send(data interface{}) (err error) {
	// 数据转换
	td, _ := data.(transportationData)
	// 读操作
	if td.Mode == contracts.ReadMode {
		err = c.sendReadMode(td)
	} else {
		err = c.sendWriteMode(td)
	}
	return
}

func (c *connector) sendWriteMode(td transportationData) error {
	ext, err := getExtendProps(td.DeviceName, td.PointName)
	if err != nil {
		return err
	}
	registerType := ext.RegisterType
	values := td.Values
	targetValues := make([]uint16, 0)
	for idx, value := range values {
		targetValue := value.Value
		// 线圈直接写  寄存器需要注意掩码值
		mask := value.Mask
		if mask != 0 &&
			(registerType == string(HoldingRegister) ||
				registerType == string(InputRegister) &&
					mask != 0xFFFF) {
			originalValue, err := c.read(td.SlaveID, registerType, ext.StartAddress+uint16(idx), 1)
			if err != nil {
				return err
			}
			targetValue = (originalValue[0] & (^mask)) | (value.Value & mask)
		}
		targetValues = append(targetValues, targetValue)
	}
	err = c.write(td.SlaveID, registerType, ext.StartAddress, targetValues)
	if err != nil {
		return err
	}
	return nil
}

func (c *connector) sendReadMode(td transportationData) error {
	totalPointRawValues := make([]pointRawValue, 0)
	ext, err := getExtendProps(td.DeviceName, td.PointName)
	if err != nil {
		return err
	}
	rawValues, err := c.read(td.SlaveID, ext.RegisterType, ext.StartAddress, ext.Quantity)
	if err != nil {
		return err
	}
	var pointNames []string
	// 虚拟点位进行拆分
	if ext.Virtual {
		pointNames = ext.Points
	} else {
		pointNames = []string{td.PointName}
	}
	totalPointRawValues, err = sliceToPointRawValue(rawValues, ext.StartAddress, td.DeviceName, pointNames)
	if err != nil {
		return err
	}
	raw := rawData{
		DeviceName:     td.DeviceName,
		PointRawValues: totalPointRawValues,
	}
	_, err = c.plugin.callback(c.plugin, raw)
	if err != nil {
		return err
	}
	return nil
}

func sliceToPointRawValue(rawValues []uint16, startAddress uint16, deviceName string, pointNames []string) ([]pointRawValue, error) {
	pointRawValues := make([]pointRawValue, len(pointNames))
	//fmt.Printf("%+v\n", rawValues)
	for idx, pointName := range pointNames {
		//fmt.Printf("slicing %s", ppd.PointName)
		ext, err := getExtendProps(deviceName, pointName)
		if err != nil {
			return nil, err
		}
		//fmt.Printf("start address %d length %d\n", ppd.Address, ppd.Quantity)
		values := rawValues[ext.StartAddress-startAddress : ext.StartAddress+ext.Quantity-startAddress]
		prv := &pointRawValue{
			Name:      pointName,
			RawValues: values,
		}
		err = prv.uint16sToValue(values, ext.RegisterType, ext.RawType, ext.ByteSwap, ext.WordSwap)
		if err != nil {
			return nil, err
		}
		pointRawValues[idx] = *prv
	}
	return pointRawValues, nil
}

func getExtendProps(deviceName, pointName string) (ext extendProps, err error) {
	point, ok := helper.CoreCache.GetPointByDevice(deviceName, pointName)
	if !ok {
		err = fmt.Errorf("point %s not found for device %s", pointName, deviceName)
	}
	err = helper.Map2Struct(point.Extends, &ext)
	address, registerType, err := castStartingAddress(point.Extends["startAddress"])
	if err != nil {
		return
	}
	ext.StartAddress = address
	if registerType != "" {
		ext.RegisterType = string(registerType)
	}
	if err != nil {
		return
	}
	return
}

// castStartingAddress modbus 地址转换
func castStartingAddress(i interface{}) (address uint16, registerType primaryTable, err error) {
	s := cast.ToString(i)
	if strings.HasPrefix(s, "0x") { //check hex format
		addr, err := strconv.ParseInt(s[2:], 16, 32)
		if err != nil {
			return 0, "", err
		}
		return cast.ToUint16(addr), "", nil
	} else if len(s) == 5 { //handle modbus format
		res, err := cast.ToUint16E(s)
		if err != nil {
			return 0, "", err
		}
		if res > 0 && res < 10000 {
			return res - 1, Coil, nil
		} else if res > 10000 && res < 20000 {
			return res - 10001, DiscreteInput, nil
		} else if res > 30000 && res < 40000 {
			return res - 30001, InputRegister, nil
		} else if res > 40000 && res < 50000 {
			return res - 40001, HoldingRegister, nil
		} else {
			return 0, "", err
		}
	}

	res, err := cast.ToUint16E(i)
	if err != nil {
		return 0, "", err
	}
	return res, "", nil
}

func (prv *pointRawValue) uint16sToValue(originalData []uint16, registerType string, rawType string,
	byteSwap bool, wordSwap bool) error {
	e := Endianness(byteSwap)
	w := WordOrder(wordSwap)
	bytes := Uint16sToBytes(e, originalData)
	length := len(originalData)
	switch strings.ToUpper(registerType) {
	case string(Coil), string(DiscreteInput):
		if length != 1 {
			return fmt.Errorf("illegal length %s", prv.Name)
		}
		prv.Value = originalData[0] == 1
		return nil
	case string(HoldingRegister), string(InputRegister):
		switch strings.ToUpper(rawType) {
		case strings.ToUpper(common.ValueTypeUint16):
			if length != 1 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = BytesToUint16(e, bytes)
			return nil
		case strings.ToUpper(common.ValueTypeInt16):
			if length != 1 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = cast.ToInt16(BytesToUint16(e, bytes))
			return nil
		case strings.ToUpper(common.ValueTypeUint32):
			if length != 2 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = BytesToUint32s(e, w, bytes)[0]
			return nil
		case strings.ToUpper(common.ValueTypeInt32):
			if length != 2 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = cast.ToInt32(BytesToUint32s(e, w, bytes)[0])
			return nil
		case strings.ToUpper(common.ValueTypeFloat32):
			if length != 2 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = BytesToFloat32s(e, w, bytes)[0]
			return nil
		case strings.ToUpper(common.ValueTypeUint64):
			if length != 4 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = BytesToUint64s(e, w, bytes)[0]
			return nil
		case strings.ToUpper(common.ValueTypeInt64):
			if length != 4 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = cast.ToInt64(BytesToUint64s(e, w, bytes)[0])
			return nil
		case strings.ToUpper(common.ValueTypeFloat64):
			if length != 4 {
				return fmt.Errorf("illegal length %s", prv.Name)
			}
			prv.Value = BytesToFloat64s(e, w, bytes)[0]
			return nil
		case strings.ToUpper(common.ValueTypeString):
			prv.Value = cast.ToString(bytes)
			return nil
		default:
			// 未填获取填错的情况不做处理
			return nil
		}
	default:
		return fmt.Errorf("unknown register type: %s", rawType)
	}
}

// Release 释放资源
// 不释放连接资源，经测试该包不支持频繁创建连接
func (c *connector) Release() (err error) {
	defer func() {
		c.mutex.Unlock()
	}()
	err = c.client.Close()
	if err != nil {
		return err
	}
	return
}

// connect 建立并打开连接
func (c *connector) connect() {
	url := fmt.Sprintf("%s://%s", c.config.Mode, c.config.Address)
	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:      url,
		Speed:    c.config.BaudRate,
		DataBits: c.config.DataBits,
		Parity:   c.config.Parity,
		StopBits: c.config.StopBits,
		Timeout:  time.Duration(c.config.Timeout) * time.Millisecond,
	})
	if err != nil {
		c.plugin.logger.Error("modbus connect error", zap.Error(err))
		return
	}
	c.client = client
}

// read 读操作
// 首次读取失败，将尝试重连 modbus 连接
func (c *connector) read(slaveId uint8, registerType string, address, quantity uint16) (values []uint16, err error) {
	if err = c.client.SetUnitId(slaveId); err != nil {
		return nil, err
	}
	switch strings.ToUpper(registerType) {
	case string(Coil):
		responseData, err := c.client.ReadCoils(address, quantity)
		if err != nil {
			return nil, err
		}
		values = boolSliceToUint16(responseData)
	case string(DiscreteInput):
		responseData, err := c.client.ReadDiscreteInputs(address, quantity)
		if err != nil {
			return nil, err
		}
		values = boolSliceToUint16(responseData)
	case string(InputRegister):
		responseData, err := c.client.ReadRegisters(address, quantity, modbus.INPUT_REGISTER)
		if err != nil {
			return nil, err
		}
		values = responseData
	case string(HoldingRegister):
		responseData, err := c.client.ReadRegisters(address, quantity, modbus.HOLDING_REGISTER)
		if err != nil {
			return nil, err
		}
		values = responseData
	default:
		return
	}
	return
}

func boolSliceToUint16(arr []bool) []uint16 {
	if arr == nil {
		return *new([]uint16)
	}
	result := make([]uint16, len(arr))
	for i, b := range arr {
		if b {
			result[i] = 1
		} else {
			result[i] = 0
		}
	}
	return result
}

func uint16SliceToBool(arr []uint16) []bool {
	if arr == nil {
		return *new([]bool)
	}
	result := make([]bool, len(arr))
	for i, v := range arr {
		if v == 1 {
			result[i] = true
		} else {
			result[i] = false
		}
	}
	return result
}

// write 写操作
func (c *connector) write(slaveID uint8, registerType string, address uint16, values []uint16) (err error) {
	err = c.client.SetUnitId(slaveID)
	if err != nil {
		return
	}
	switch strings.ToUpper(registerType) {
	case string(Coil):
		bools := uint16SliceToBool(values)
		err = c.client.WriteCoils(address, bools)
	case string(HoldingRegister):
		err = c.client.WriteRegisters(address, values)
	default:
		return common.UnsupportedWriteCommandRegisterType
	}
	return
}
