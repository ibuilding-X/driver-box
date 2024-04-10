package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
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

func newConnector(plugin *Plugin, cf *connectorConfig) (*connector, error) {
	freq := cf.PollFreq
	if freq == 0 {
		freq = 1000
	}
	minInterval := cf.MinInterval
	if minInterval == 0 {
		minInterval = 100
	}
	maxLen := cf.MaxLen
	if maxLen == 0 {
		maxLen = 32
	}
	retry := cf.Retry
	if retry == 0 {
		retry = 3
	}
	conn := &connector{
		plugin:      plugin,
		pollFreq:    freq,
		maxLen:      maxLen,
		minInterval: minInterval,
		retry:       3,
		virtual:     cf.Virtual || config.IsVirtual(),
	}
	conn.devices = make(map[string]*slaveDevice)
	conn.pointMap = make(map[string]map[string]*pointConfig)
	url := fmt.Sprintf("%s://%s", cf.Mode, cf.Address)
	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:      url,
		Speed:    cf.BaudRate,
		DataBits: cf.DataBits,
		Parity:   cf.Parity,
		StopBits: cf.StopBits,
		Timeout:  time.Duration(cf.Timeout) * time.Millisecond,
	})
	if err != nil {
		return nil, fmt.Errorf("modbus init connection error: %s", err.Error())
	}
	conn.client = client
	//启动采集任务
	err = conn.initCollectTask(cf)
	return conn, err
}

func (c *connector) initCollectTask(cf *connectorConfig) (err error) {
	err = c.createPointGroup(err)
	//注册定时采集任务
	duration := cf.Duration
	if duration == "" {
		helper.Logger.Warn("modbus connection duration is empty, use default 5s", zap.String("key", c.key))
		duration = "5s"
	}
	future, err := helper.Crontab.AddFunc(duration, func() {
		//遍历所有通讯设备
		for unitID, device := range c.devices {
			if len(device.pointGroup) == 0 {
				helper.Logger.Warn("device has none read point", zap.String("unitID", unitID))
				continue
			}
			//批量遍历通讯设备下的点位，并将结果关联至物模型设备
			for i, group := range device.pointGroup {
				if c.close {
					helper.Logger.Warn("modbus connection is closed, ignore collect task!", zap.String("key", c.key))
					break
				}

				if group.LatestTime.Add(group.Duration).After(time.Now()) {
					continue
				}
				//采集时间未到
				helper.Logger.Debug("timer read modbus", zap.Any("group", i), zap.Any("latestTime", group.LatestTime), zap.Any("duration", group.Duration))
				bac := command{
					mode:  plugin.ReadMode,
					value: group,
				}
				if err = c.Send(bac); err != nil {
					helper.Logger.Error("read error", zap.Error(err))
					//通讯失败，触发离线
					devices := make(map[string]interface{})
					for _, pconfig := range group.points {
						if devices[pconfig.DeviceSn] != nil {
							continue
						}
						devices[pconfig.DeviceSn] = pconfig.PointName
						_ = helper.DeviceShadow.MayBeOffline(pconfig.DeviceSn)
					}
				}
				group.LatestTime = time.Now()
			}

		}
	})
	if err != nil {
		return err
	} else {
		c.collectTask = future
		return nil
	}
}

// 采集任务分组
func (c *connector) createPointGroup(err error) error {
	for _, model := range c.plugin.config.DeviceModels {
		for _, dev := range model.Devices {
			if dev.ConnectionKey != c.key {
				continue
			}
			for _, point := range model.DevicePoints {
				p := point.ToPoint()
				if p.ReadWrite != config.ReadWrite_R && p.ReadWrite != config.ReadWrite_RW {
					continue
				}
				var ext pointConfig
				if err = helper.Map2Struct(p.Extends, &ext); err != nil {
					helper.Logger.Error("error bacnet config", zap.Any("config", p.Extends), zap.Error(err))
					continue
				}
				//未设置，则默认每秒采集一次
				if ext.Duration == "" {
					ext.Duration = "1s"
				}
				duration, err := time.ParseDuration(ext.Duration)
				if err != nil {
					helper.Logger.Error("error bacnet duration config", zap.String("deviceSn", dev.DeviceSn), zap.Any("config", p.Extends), zap.Error(err))
					duration = time.Second
				}

				device, err := c.createDevice(dev.Properties)
				ok := false
				for _, group := range device.pointGroup {
					//相同采集频率为同一组
					if group.Duration != duration {
						continue
					}
					//不同寄存器类型不为一组
					if group.RegisterType != ext.RegisterType {
						continue
					}
					//
					//当前点位已存在
					for _, obj := range group.points {
						if obj.Address == ext.Address {
							obj.DeviceSn = dev.DeviceSn
							obj.PointName = p.Name
							ok = true
							break
						}
					}
					if ok {
						break
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
					if end-start <= c.maxLen {
						group.points = append(group.points, &ext)
						ok = true
						group.Address = start
						group.Quantity = end - start
						break
					}
				}
				//新增一个点位组
				if !ok {
					ext.DeviceSn = dev.DeviceSn
					ext.PointName = p.Name
					device.pointGroup = append(device.pointGroup, &pointGroup{
						unitID:   device.unitID,
						Duration: duration,
						points: []*pointConfig{
							&ext,
						},
					})
				}
			}
		}
	}
	return err
}

// Send 发送数据
func (c *connector) Send(data interface{}) (err error) {
	cmd := data.(command)
	switch cmd.mode {
	// 读
	case plugin.ReadMode:
		group := cmd.value.(pointGroup)
		err = c.sendReadCommand(group)
	case plugin.WriteMode:
		value := cmd.value.(writeValue)
		err = c.sendWriteCommand(value)
	default:
		return common.NotSupportMode
	}

	return
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

type primaryTable string

const (
	Coil            primaryTable = "COIL"             // 线圈
	DiscreteInput   primaryTable = "DISCRETE_INPUT"   // 离散输入
	InputRegister   primaryTable = "INPUT_REGISTER"   // 离散寄存器
	HoldingRegister primaryTable = "HOLDING_REGISTER" // 保持寄存器
)

func getPrimaryTable(registerType string) *primaryTable {
	for _, pt := range []primaryTable{Coil, DiscreteInput, InputRegister, HoldingRegister} {
		if registerType == string(pt) {
			return &pt
		}
	}
	return nil
}

type taskGroup struct {
	slaveId      uint8          // 从机地址
	registerType primaryTable   // 寄存器类型
	address      uint16         // 起始地址
	length       uint16         // 长度
	points       []*pointConfig // 当前的分组所包含的所有点位
}

func newTaskGroup(slaveId uint8, point *pointConfig) *taskGroup {
	return &taskGroup{
		slaveId:      slaveId,
		registerType: point.RegisterType,
		address:      point.Address,
		length:       point.Quantity,
		points:       []*pointConfig{point},
	}
}

func (tg *taskGroup) addPoint(point *pointConfig) {
	tg.points = append(tg.points, point)
}

func isWritable(rw string) bool {
	return strings.Contains(strings.ToUpper(rw), "W")
}

func (c *connector) ensureFreq() {
	nt := c.lastReq.Add(time.Duration(c.pollFreq) * time.Millisecond)
	if time.Now().Before(nt) {
		time.Sleep(time.Until(nt))
	}
	c.lastPoll = time.Now()
}

func (c *connector) ensureInterval() {
	np := c.lastReq.Add(time.Duration(c.minInterval) * time.Millisecond)
	if time.Now().Before(np) {
		time.Sleep(time.Until(np))
	}
}

func (c *connector) sendReadCommand(group pointGroup) error {
	err := c.client.Open()
	if err != nil {
		return err
	}
	defer func() {
		err = c.client.Close()
		if err != nil {
			helper.Logger.Error(fmt.Sprintf("close plugin error: %v", err))
		}
	}()
	values, err := c.read(group.unitID, string(group.RegisterType), group.Address, group.Quantity)
	if err != nil {
		return err
	}
	// 转化数据并上报
	for _, point := range group.points {
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
				c.plugin.logger.Error(fmt.Sprintf("unsupported raw type: %v", point))
				continue
			}
		}
		pointReadValue := plugin.PointReadValue{
			SN:        point.DeviceSn,
			PointName: point.PointName,
			Value:     value,
		}
		_, err = callback.OnReceiveHandler(c.plugin, pointReadValue)
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
func getBitsFromPosition(num uint16, startPos, bitCount int) uint16 {
	// 将指定位置后的位清零
	mask := uint16(((1 << bitCount) - 1) << startPos)
	num = num & mask
	num = num >> startPos
	return num
}

func mergeBitsIntoUint16(num, startPos, bitCount int, regValue uint16) uint16 {
	// 创建掩码，用于清除要替换的位
	mask := uint16((1<<bitCount)-1) << startPos

	// 清除要替换的位
	clearedValue := regValue &^ mask

	// 将v的二进制表示左移i位，然后与清除后的值进行按位或操作
	replacedValue := clearedValue | uint16(num<<startPos)

	return replacedValue
}

type PointValue struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func (c *connector) sendWriteCommand(pc writeValue) error {

	var values []uint16
	value := pc.Value
	switch pc.RegisterType {
	case Coil: // 线圈固定长度1
		i, err := helper.Conv2Int64(value)
		if err != nil {
			return fmt.Errorf("convert cmd  error: %v", err)
		}
		values = []uint16{uint16(i & 1)}
	case HoldingRegister:
		valueStr := fmt.Sprintf("%v", value)
		switch strings.ToUpper(pc.RawType) {
		case strings.ToUpper(common.ValueTypeUint16):
			v, err := strconv.ParseUint(valueStr, 10, 16)
			if err != nil {
				return fmt.Errorf("convert value %v to uint16 error: %v", value, err)
			}
			// TODO: 位写
			if pc.BitLen > 0 {
				if v > (1<<pc.BitLen - 1) {
					return fmt.Errorf("too large value %v to set in %d bits", v, pc.BitLen)
				}
				c.ensureInterval()
				uint16s, err := c.read(pc.unitID, string(pc.RegisterType), pc.Address, pc.Quantity)
				uint16Val := uint16s[0]
				if pc.ByteSwap {
					uint16Val = (uint16Val << 8) | (uint16Val >> 8)
				}
				c.lastReq = time.Now()
				if err != nil {
					return fmt.Errorf("read original register error: %v", err)
				}
				intoUint16 := mergeBitsIntoUint16(int(v), pc.Bit, pc.BitLen, uint16Val)
				if pc.ByteSwap {
					intoUint16 = (intoUint16 << 8) | (intoUint16 >> 8)
				}
				values = []uint16{intoUint16}
				break
			}
			out := make([]byte, 2)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint16(out, uint16(v))
			} else {
				binary.BigEndian.PutUint16(out, uint16(v))
			}
			values = []uint16{binary.BigEndian.Uint16(out)}
		case strings.ToUpper(common.ValueTypeInt16):
			v, err := strconv.ParseInt(valueStr, 10, 16)
			if err != nil {
				return fmt.Errorf("convert value %v to int16 error: %v", value, err)
			}
			if pc.BitLen > 0 {
				if v > (1<<pc.BitLen - 1) {
					return fmt.Errorf("too large value %v to set in %d bits", v, pc.BitLen)
				} else if v < 0 {
					return fmt.Errorf("negative value %v not allowed to set in bits", v)
				}
				c.ensureInterval()
				uint16s, err := c.read(pc.unitID, string(pc.RegisterType), pc.Address, pc.Quantity)
				uint16Val := uint16s[0]
				if pc.ByteSwap {
					uint16Val = (uint16Val << 8) | (uint16Val >> 8)
				}
				c.lastReq = time.Now()
				if err != nil {
					return fmt.Errorf("read original register error: %v", err)
				}
				intoUint16 := mergeBitsIntoUint16(int(v), pc.Bit, pc.BitLen, uint16Val)
				if pc.ByteSwap {
					intoUint16 = (intoUint16 << 8) | (intoUint16 >> 8)
				}
				values = []uint16{intoUint16}
				break
			}
			out := make([]byte, 2)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint16(out, uint16(v))
			} else {
				binary.BigEndian.PutUint16(out, uint16(v))
			}
			values = []uint16{binary.BigEndian.Uint16(out)}
		case strings.ToUpper(common.ValueTypeUint32):
			v, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return fmt.Errorf("convert value %v to uint32 error: %v", value, err)
			}
			out := make([]byte, 4)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint32(out, uint32(v))
			} else {
				binary.BigEndian.PutUint32(out, uint32(v))
			}
			if pc.WordSwap {
				out[0], out[1], out[2], out[3] =
					out[2], out[3], out[0], out[1]
			}
			values = []uint16{binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]})}
		case strings.ToUpper(common.ValueTypeInt32):
			v, err := strconv.ParseInt(valueStr, 10, 32)
			if err != nil {
				return fmt.Errorf("convert value %v to int32 error: %v", value, err)
			}
			out := make([]byte, 4)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint32(out, uint32(v))
			} else {
				binary.BigEndian.PutUint32(out, uint32(v))
			}
			if pc.WordSwap {
				out[0], out[1], out[2], out[3] =
					out[2], out[3], out[0], out[1]
			}
			values = []uint16{binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]})}
		case strings.ToUpper(common.ValueTypeFloat32):
			v, err := strconv.ParseFloat(valueStr, 32)
			if err != nil {
				return fmt.Errorf("convert value %v to float32 error: %v", value, err)
			}
			v32 := float32(v)
			bits := math.Float32bits(v32)
			out := make([]byte, 4)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint32(out, bits)
			} else {
				binary.BigEndian.PutUint32(out, bits)
			}
			if pc.WordSwap {
				out[0], out[1], out[2], out[3] =
					out[2], out[3], out[0], out[1]
			}
			values = []uint16{binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]})}
		case strings.ToUpper(common.ValueTypeUint64):
			v, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("convert value %v to uint64 error: %v", value, err)
			}
			out := make([]byte, 8)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint64(out, v)
			} else {
				binary.BigEndian.PutUint64(out, v)
			}
			if pc.WordSwap {
				out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7] =
					out[2], out[3], out[0], out[1], out[6], out[7], out[4], out[5]
			}
			values = []uint16{
				binary.BigEndian.Uint16([]byte{out[6], out[7]}),
				binary.BigEndian.Uint16([]byte{out[4], out[5]}),
				binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]}),
			}
		case strings.ToUpper(common.ValueTypeInt64):
			v, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("convert value %v to int64 error: %v", value, err)
			}
			out := make([]byte, 8)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint64(out, uint64(v))
			} else {
				binary.BigEndian.PutUint64(out, uint64(v))
			}
			if pc.WordSwap {
				out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7] =
					out[2], out[3], out[0], out[1], out[6], out[7], out[4], out[5]
			}
			values = []uint16{
				binary.BigEndian.Uint16([]byte{out[6], out[7]}),
				binary.BigEndian.Uint16([]byte{out[4], out[5]}),
				binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]}),
			}
		case strings.ToUpper(common.ValueTypeFloat64):
			v, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				return fmt.Errorf("convert value %v to float64 error: %v", value, err)
			}
			out := make([]byte, 8)
			if pc.ByteSwap {
				binary.LittleEndian.PutUint64(out, math.Float64bits(v))
			} else {
				binary.BigEndian.PutUint64(out, math.Float64bits(v))
			}
			if pc.WordSwap {
				out[0], out[1], out[2], out[3], out[4], out[5], out[6], out[7] =
					out[2], out[3], out[0], out[1], out[6], out[7], out[4], out[5]
			}
			values = []uint16{
				binary.BigEndian.Uint16([]byte{out[6], out[7]}),
				binary.BigEndian.Uint16([]byte{out[4], out[5]}),
				binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]}),
			}
		case strings.ToUpper(common.ValueTypeString):
			valueBytes := []byte(valueStr)
			if len(valueBytes) > int(pc.Quantity*2) {
				return fmt.Errorf("too long string [%v] to set in %d registers", valueStr, pc.Quantity)
			}
			if pc.ByteSwap {
				for i := 0; i < len(valueBytes); i += 2 {
					if i+1 < len(valueBytes) {
						valueBytes[i], valueBytes[i+1] = valueBytes[i+1], valueBytes[i]
					}
				}
			}
			if pc.WordSwap {
				for i := 0; i < len(valueBytes); i += 4 {
					if i+3 < len(valueBytes) {
						valueBytes[i], valueBytes[i+1], valueBytes[i+2], valueBytes[i+3] =
							valueBytes[i+2], valueBytes[i+3], valueBytes[i], valueBytes[i+1]
					}
				}
			}
			values = make([]uint16, pc.Quantity)
			for i := 0; i < len(valueBytes); i += 2 {
				if i+1 < len(valueBytes) {
					values[i/2] = binary.BigEndian.Uint16(valueBytes[i : i+2])
				} else {
					values[i/2] = binary.BigEndian.Uint16([]byte{valueBytes[i], 0})
				}
			}
			for i := 0; i < len(values)/2; i++ {
				values[i], values[len(values)-1-i] = values[len(values)-1-i], values[i]
			}
		default:
			return fmt.Errorf("unsupported raw type: %v", pc)
		}
	default:
		return fmt.Errorf("unsupported write register type: %v", pc)
	}
	c.ensureInterval()
	var err error
	for i := 0; i < c.retry; i++ {
		if err = c.write(pc.unitID, pc.RegisterType, pc.Address, values); err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("write [%v] error: %v", pc, err)
	}
	c.lastReq = time.Now()
	return nil
}

func divideStrings(str1, str2 string) (string, error) {
	// 验证输入是否为合法数字
	_, err := strconv.ParseFloat(str1, 64)
	if err != nil {
		return "", fmt.Errorf("第一个字符串不是合法的数字")
	}

	_, err = strconv.ParseFloat(str2, 64)
	if err != nil {
		return "", fmt.Errorf("第二个字符串不是合法的数字")
	}

	// 将字符串转换为浮点数进行除法运算
	num1, _ := strconv.ParseFloat(str1, 64)
	num2, _ := strconv.ParseFloat(str2, 64)
	result := num1 / num2

	// 将结果转换为字符串
	resultStr := strconv.FormatFloat(result, 'f', -1, 64)

	return resultStr, nil
}

func reverseUint16s(in []uint16) {
	for i, j := 0, len(in)-1; i < j; i, j = i+1, j-1 {
		in[i], in[j] = in[j], in[i]
	}
}

// castModbusAddress modbus 地址转换
func castModbusAddress(i interface{}) (address uint16, registerType primaryTable, err error) {
	s := cast.ToString(i)
	if strings.HasPrefix(s, "0x") { //check hex format
		addr, err := strconv.ParseInt(s[2:], 16, 32)
		if err != nil {
			return 0, "", err
		}
		return cast.ToUint16(addr), "", nil
	} else if strings.HasSuffix(s, "d") {
		addr, err := strconv.Atoi(strings.ReplaceAll(s, "d", ""))
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
		}
	}

	res, err := cast.ToUint16E(i)
	if err != nil {
		return 0, "", err
	}
	return res, "", nil
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
		return nil, fmt.Errorf("unsupported register type %v", registerType)
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
	err = c.client.SetUnitId(slaveID)
	if err != nil {
		return
	}
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

func convToPointConfig(extends map[string]interface{}) (*pointConfig, error) {
	pc := new(pointConfig)
	err := helper.Map2Struct(extends, pc)
	if err != nil {
		return nil, fmt.Errorf("convert %v to point config error: %s", extends, err.Error())
	}
	startAddress, ok := extends["startAddress"]
	if !ok {
		return nil, fmt.Errorf("start address missed")
	}
	address, registerType, err := castModbusAddress(startAddress)
	if err != nil {
		return nil, fmt.Errorf("convert start address error: %s", err.Error())
	}
	pc.Address = address
	if registerType != "" {
		pc.RegisterType = registerType
	}
	if getPrimaryTable(string(pc.RegisterType)) == nil {
		return nil, fmt.Errorf("incorrect register type: %s", pc.RegisterType)
	}
	switch pc.RegisterType {
	case Coil, DiscreteInput: // 线圈及离散输入仅支持读一个
		pc.Quantity = 1
	case InputRegister, HoldingRegister:
		switch strings.ToUpper(pc.RawType) {
		case strings.ToUpper(common.ValueTypeUint16), strings.ToUpper(common.ValueTypeInt16):
			pc.Quantity = 1
		case strings.ToUpper(common.ValueTypeUint32), strings.ToUpper(common.ValueTypeInt32), strings.ToUpper(common.ValueTypeFloat32):
			pc.Quantity = 2
		case strings.ToUpper(common.ValueTypeUint64), strings.ToUpper(common.ValueTypeInt64), strings.ToUpper(common.ValueTypeFloat64):
			pc.Quantity = 4
		case strings.ToUpper(common.ValueTypeString):
			if pc.Quantity == 0 {
				pc.Quantity = 1
			}
		default:
			return nil, fmt.Errorf("unsupported raw type %s", pc.RawType)
		}
	default:
		return nil, fmt.Errorf("unsuported register type: %s", pc.RegisterType)
	}

	if pc.Quantity == 0 {
		pc.Quantity = 1
	}

	return pc, nil
}
func (c *connector) createDevice(properties map[string]string) (d *slaveDevice, err error) {
	var dp deviceProtocol
	if err = helper.Map2Struct(properties, &dp); err != nil {
		return nil, err
	}

	if len(dp.unitID) == 0 {
		return nil, errors.New("none unitID")
	}
	uintIdVal, err := strconv.ParseUint(dp.unitID, 10, 8)

	if err != nil {
		return nil, err
	}

	var group []*pointGroup
	d = &slaveDevice{
		unitID:     uint8(uintIdVal),
		pointGroup: group,
	}
	c.devices[dp.unitID] = d
	return d, nil
}
