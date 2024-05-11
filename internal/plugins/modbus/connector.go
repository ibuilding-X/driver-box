package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/simonvetter/modbus"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"math"
	"strconv"
	"strings"
	"time"
)

func newConnector(p *Plugin, cf *ConnectionConfig) (*connector, error) {
	if cf.MinInterval == 0 {
		cf.MinInterval = 100
	}
	if cf.Retry == 0 {
		cf.Retry = 3
	}
	if cf.Timeout <= 0 {
		cf.Timeout = 1000
	}
	if cf.BatchReadLen == 0 {
		cf.BatchReadLen = 32
	}

	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:      fmt.Sprintf("%s://%s", cf.Mode, cf.Address),
		Speed:    cf.BaudRate,
		DataBits: cf.DataBits,
		Parity:   cf.Parity,
		StopBits: cf.StopBits,
		Timeout:  time.Duration(cf.Timeout) * time.Millisecond,
	})
	conn := &connector{
		Connection: plugin.Connection{
			Ls:           p.ls,
			ScriptEnable: helper.ScriptExists(p.config.Key),
		},
		config:  cf,
		plugin:  p,
		client:  client,
		virtual: cf.Virtual || config.IsVirtual(),
		devices: make(map[uint8]*slaveDevice),
	}
	return conn, err
}

func (c *connector) initCollectTask(conf *ConnectionConfig) (*crontab.Future, error) {
	if !conf.Enable {
		return nil, errors.New("modbus connection is disabled, ignore collect task")
	}
	if len(c.devices) == 0 {
		return nil, errors.New("no device to collect")
	}
	//注册定时采集任务
	return helper.Crontab.AddFunc("1s", func() {
		//遍历所有通讯设备
		for unitID, device := range c.devices {
			if len(device.pointGroup) == 0 {
				helper.Logger.Warn("device has none read point", zap.Uint8("unitID", unitID))
				continue
			}
			//批量遍历通讯设备下的点位，并将结果关联至物模型设备
			for i, group := range device.pointGroup {
				if c.close {
					helper.Logger.Warn("modbus connection is closed, ignore collect task!", zap.String("key", c.ConnectionKey))
					break
				}

				//采集时间未到
				if group.LatestTime.Add(group.Duration).After(time.Now()) {
					continue
				}

				helper.Logger.Debug("timer read modbus", zap.Any("group", i), zap.Any("latestTime", group.LatestTime), zap.Any("duration", group.Duration))
				bac := command{
					Mode:  plugin.ReadMode,
					Value: group,
				}
				if err := c.Send(bac); err != nil {
					helper.Logger.Error("read error", zap.Any("connection", conf), zap.Any("group", group), zap.Error(err))
					//通讯失败，触发离线
					devices := make(map[string]interface{})
					for _, point := range group.Points {
						if devices[point.DeviceId] != nil {
							continue
						}
						devices[point.DeviceId] = point.Name
						_ = helper.DeviceShadow.MayBeOffline(point.DeviceId)
					}
				}
				group.LatestTime = time.Now()
			}

		}
	})
}

// 采集任务分组
func (c *connector) createPointGroup(conf *ConnectionConfig, model config.DeviceModel, dev config.Device) {
	groupIndex := 0
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

		device, err := c.createDevice(dev.Properties)
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
				index:        groupIndex,
				UnitID:       device.unitID,
				Duration:     duration,
				RegisterType: ext.RegisterType,
				Address:      ext.Address,
				Quantity:     ext.Quantity,
				Points: []*Point{
					ext,
				},
			})
			groupIndex++
		}
	}

}

// ProtocolAdapter 协议适配器
func (p *connector) ProtocolAdapter() plugin.ProtocolAdapter {
	return p
}

// Send 发送数据
func (c *connector) Send(data interface{}) (err error) {
	cmd := data.(command)
	switch cmd.Mode {
	// 读
	case plugin.ReadMode:
		group := cmd.Value.(*pointGroup)
		err = c.sendReadCommand(group)
	case plugin.WriteMode:
		values := cmd.Value.([]*writeValue)
		for _, value := range values {
			err = c.sendWriteCommand(value)
		}
	case BatchReadMode:
		groups := cmd.Value.([]*pointGroup)
		for _, group := range groups {
			err = c.sendReadCommand(group)
			if err != nil {
				return err
			}
		}
	default:
		return common.NotSupportMode
	}

	return
}

// Release 释放资源
// 不释放连接资源，经测试该包不支持频繁创建连接
func (c *connector) Release() (err error) {
	return
}

func (c *connector) Close() {
	c.close = true
	if c.collectTask != nil {
		c.collectTask.Disable()
	}
}

// ensureInterval 确保与前一次IO至少间隔minInterval毫秒
func (c *connector) ensureInterval() {
	np := c.latestIoTime.Add(time.Duration(c.config.MinInterval) * time.Millisecond)
	if time.Now().Before(np) {
		time.Sleep(time.Until(np))
	}
	c.latestIoTime = time.Now()
}

func (c *connector) sendReadCommand(group *pointGroup) error {
	values, err := c.read(group.UnitID, string(group.RegisterType), group.Address, group.Quantity)
	if err != nil {
		return err
	}
	// 转化数据并上报
	for _, point := range group.Points {
		var value interface{}
		start := point.Address - group.Address
		rawValues := values[start : start+point.Quantity]
		reverseUint16s(rawValues)
		switch point.RegisterType {
		case Coil, DiscreteInput: // 线圈和离散都是单个长度，直接返回值即可
			value = rawValues[0]
		case InputRegister, HoldingRegister: // 输入寄存器和保持寄存器需要根据大小端还有bit位进行处理
			switch strings.ToUpper(point.RawType) {
			case strings.ToUpper(common.ValueTypeUint16), strings.ToUpper(common.ValueTypeInt16):
				out := getBytesFromUint16s(rawValues, point.ByteSwap)
				val := binary.BigEndian.Uint16(out)
				// 根据bit位读取数据
				if point.BitLen > 0 {
					value = getBitsFromPosition(val, point.Bit, point.BitLen)
				} else {
					if strings.ToUpper(point.RawType) == strings.ToUpper(common.ValueTypeInt16) {
						value = int16(val)
					} else {
						value = val
					}
				}
			case strings.ToUpper(common.ValueTypeUint32), strings.ToUpper(common.ValueTypeInt32), strings.ToUpper(common.ValueTypeFloat32):
				out := getBytesFromUint16s(rawValues, point.ByteSwap)
				out = swapWords(out, point.WordSwap)
				val := binary.BigEndian.Uint32(out)
				switch strings.ToUpper(point.RawType) {
				case strings.ToUpper(common.ValueTypeUint32):
					value = val
				case strings.ToUpper(common.ValueTypeInt32):
					value = int32(val)
				case strings.ToUpper(common.ValueTypeFloat32):
					value = math.Float32frombits(val)
				}
			case strings.ToUpper(common.ValueTypeUint64), strings.ToUpper(common.ValueTypeInt64), strings.ToUpper(common.ValueTypeFloat64):
				out := getBytesFromUint16s(rawValues, point.ByteSwap)
				out = swapWords(out, point.WordSwap)
				val := binary.BigEndian.Uint64(out)
				switch strings.ToUpper(point.RawType) {
				case strings.ToUpper(common.ValueTypeUint64):
					value = val
				case strings.ToUpper(common.ValueTypeInt64):
					value = int32(val)
				case strings.ToUpper(common.ValueTypeFloat64):
					value = math.Float64frombits(val)
				}
			case strings.ToUpper(common.ValueTypeString):
				out := getBytesFromUint16s(rawValues, point.ByteSwap)
				out = swapWords(out, point.WordSwap)
				value = string(out)
			default:
				helper.Logger.Error(fmt.Sprintf("unsupported raw type: %v", point))
				continue
			}
		}
		pointReadValue := plugin.PointReadValue{
			ID:        point.DeviceId,
			PointName: point.Name,
			Value:     value,
		}
		_, err = callback.OnReceiveHandler(c, pointReadValue)
		if err != nil {
			helper.Logger.Error("error modbus callback", zap.Any("data", pointReadValue), zap.Error(err))
		}
	}
	return nil
}

func getBytesFromUint16s(values []uint16, byteSwap bool) (out []byte) {
	for _, value := range values {
		bytes := make([]byte, 2)
		if byteSwap {
			binary.LittleEndian.PutUint16(bytes, value)
		} else {
			binary.BigEndian.PutUint16(bytes, value)
		}
		out = append(out, bytes...)
	}
	return
}

func swapWords(in []byte, wordSwap bool) (out []byte) {
	if len(in) >= 4 {
		for i := 0; i < len(in); i += 4 {
			if wordSwap {
				out = append(out, []byte{
					in[i+2], in[i+3], in[i], in[i+1],
				}...)
			} else {
				out = append(out, []byte{
					in[i], in[i+1], in[i+2], in[i+3],
				}...)
			}
		}
	} else {
		out = in
	}
	return
}

// 获取从指定位置开始的指定位数的值
func getBitsFromPosition(num uint16, startPos, bitCount uint8) uint16 {
	// 将指定位置后的位清零
	mask := uint16(((1 << bitCount) - 1) << startPos)
	num = num & mask
	num = num >> startPos
	return num
}

func mergeBitsIntoUint16(num int, startPos, bitCount uint8, regValue uint16) uint16 {
	// 创建掩码，用于清除要替换的位
	mask := uint16((1<<bitCount)-1) << startPos

	// 清除要替换的位
	clearedValue := regValue &^ mask

	// 将v的二进制表示左移i位，然后与清除后的值进行按位或操作
	replacedValue := clearedValue | uint16(num<<startPos)

	return replacedValue
}

func (c *connector) sendWriteCommand(pc *writeValue) error {
	var err error
	for i := 0; i < c.config.Retry; i++ {
		if err = c.write(pc.unitID, pc.RegisterType, pc.Address, pc.Value); err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("write [%v] error: %v", pc, err)
	}
	return nil
}

func reverseUint16s(in []uint16) {
	for i, j := 0, len(in)-1; i < j; i, j = i+1, j-1 {
		in[i], in[j] = in[j], in[i]
	}
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

// read 读操作
// 首次读取失败，将尝试重连 modbus 连接
func (c *connector) read(slaveId uint8, registerType string, address, quantity uint16) (values []uint16, err error) {
	err = c.openModbusClient()
	if err != nil {
		return nil, err
	}
	defer c.closeModbusClient()
	if err = c.client.SetUnitId(slaveId); err != nil {
		return nil, err
	}
	c.ensureInterval()
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
		return nil, fmt.Errorf("unsupported register type %v", registerType)
	}
	return
}

func (c *connector) openModbusClient() error {
	c.mutex.Lock()
	err := c.client.Open()
	if err != nil {
		c.mutex.Unlock()
	}
	return err
}

func (c *connector) closeModbusClient() {
	defer func() {
		c.mutex.Unlock()
	}()
	_ = c.client.Close()
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
		return []bool{}
	}
	result := make([]bool, len(arr))
	for i, v := range arr {
		result[i] = v&1 == 1
	}
	return result
}

// write 写操作
func (c *connector) write(slaveID uint8, registerType primaryTable, address uint16, values []uint16) (err error) {
	err = c.openModbusClient()
	if err != nil {
		return err
	}
	defer c.closeModbusClient()
	err = c.client.SetUnitId(slaveID)
	if err != nil {
		return
	}
	c.ensureInterval()
	switch registerType {
	case Coil:
		bools := uint16SliceToBool(values)
		err = c.client.WriteCoils(address, bools)
	case HoldingRegister:
		err = c.client.WriteRegisters(address, values)
	default:
		return common.UnsupportedWriteCommandRegisterType
	}
	return
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

func (c *connector) createDevice(properties map[string]string) (d *slaveDevice, err error) {
	unitID, err := getUnitId(properties)
	d, ok := c.devices[unitID]
	if ok {
		return d, nil
	}

	var group []*pointGroup
	d = &slaveDevice{
		unitID:     unitID,
		pointGroup: group,
	}
	c.devices[unitID] = d
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
