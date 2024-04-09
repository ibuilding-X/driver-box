package modbus

import (
	"encoding/binary"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/simonvetter/modbus"
	"github.com/spf13/cast"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// connectorConfig 连接器配置
type connectorConfig struct {
	Address     string `json:"address"`     // 地址：例如：127.0.0.1:502
	Mode        string `json:"mode"`        // 连接模式：rtuovertcp、rtu
	BaudRate    uint   `json:"baudRate"`    // 波特率（仅串口模式）
	DataBits    uint   `json:"dataBits"`    // 数据位（仅串口模式）
	StopBits    uint   `json:"stopBits"`    // 停止位（仅串口模式）
	Parity      uint   `json:"parity"`      // 奇偶性校验（仅串口模式）
	PollFreq    uint64 `json:"pollFreq"`    // 读取周期，
	MaxLen      uint16 `json:"maxLen"`      // 最长连续读个数
	MinInterval uint   `json:"minInterval"` // 最小读取间隔
	Timeout     int    `json:"timeout"`     // 请求超时
	Retry       int    `json:"retry"`       // 重试次数
}

func newConnector(plugin *Plugin, config *connectorConfig) (*connector, error) {
	freq := config.PollFreq
	if freq == 0 {
		freq = 1000
	}
	minInterval := config.MinInterval
	if minInterval == 0 {
		minInterval = 100
	}
	maxLen := config.MaxLen
	if maxLen == 0 {
		maxLen = 32
	}
	retry := config.Retry
	if retry == 0 {
		retry = 3
	}
	conn := &connector{
		plugin:      plugin,
		pollFreq:    freq,
		maxLen:      maxLen,
		minInterval: minInterval,
		retry:       3,
	}
	conn.devices = make(map[uint8]map[primaryTable][]*pointConfig)
	conn.pointMap = make(map[string]map[string]*pointConfig)
	url := fmt.Sprintf("%s://%s", config.Mode, config.Address)
	client, err := modbus.NewClient(&modbus.ClientConfiguration{
		URL:      url,
		Speed:    config.BaudRate,
		DataBits: config.DataBits,
		Parity:   config.Parity,
		StopBits: config.StopBits,
		Timeout:  time.Duration(config.Timeout) * time.Millisecond,
	})
	if err != nil {
		return nil, fmt.Errorf("modbus init connection error: %s", err.Error())
	}
	conn.client = client
	return conn, nil
}

// connector 连接器
type connector struct {
	plugin      *Plugin
	client      *modbus.ModbusClient
	maxLen      uint16    // 最长连续读个数
	minInterval uint      // 读取间隔
	polling     bool      // 执行轮询
	pollFreq    uint64    // 轮询间隔
	lastPoll    time.Time // 上次轮询
	lastReq     time.Time // 上次执行
	mutex       sync.Mutex
	devices     map[uint8]map[primaryTable][]*pointConfig
	pointMap    map[string]map[string]*pointConfig
	retry       int
}

// Send 发送数据
func (c *connector) Send(data interface{}) (err error) {
	err = c.sendWriteCommand(data)
	if err != nil {
		c.plugin.logger.Error(fmt.Sprintf("write %v error: %v", data, err))
	} else {
		c.plugin.logger.Info(fmt.Sprintf("write %v succeeded", data))
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

func (c *connector) startPollTasks() error {
	taskGroups := make([]*taskGroup, 0)
	for slaveId, registerGroups := range c.devices {
		for registerType, points := range registerGroups {
			pointsToRead := make([]*pointConfig, 0)
			for _, point := range points {
				if isReadable(point.ReadWrite) {
					pointsToRead = append(pointsToRead, point)
				}
			}
			sort.Slice(pointsToRead, func(i, j int) bool {
				p1 := pointsToRead[i]
				p2 := pointsToRead[j]
				return p1.Address < p2.Address
			})
			for _, pointToRead := range pointsToRead {
				ttn := len(taskGroups)
				if ttn == 0 {
					taskGroups = append(taskGroups, newTaskGroup(slaveId, pointToRead))
					continue
				}
				pre := taskGroups[ttn-1]
				if pre.slaveId == slaveId && pre.registerType == registerType &&
					pre.address+pre.length >= pointToRead.Address {
					// 相同从机、相同类型寄存器、前后连续地址相连则进行整合
					totalRegNum := pointToRead.Address + pointToRead.Quantity - pre.address
					if totalRegNum <= c.maxLen {
						pre.length = totalRegNum
						pre.addPoint(pointToRead)
					} else {
						taskGroups = append(taskGroups, newTaskGroup(slaveId, pointToRead))
					}
				} else {
					taskGroups = append(taskGroups, newTaskGroup(slaveId, pointToRead))
				}
			}
		}
	}
	c.lastPoll = time.Now()
	c.lastReq = time.Now()
	c.polling = true
	go func() {
		for c.polling {
			c.ensureFreq()
			for _, tg := range taskGroups {
				err := c.executePollTask(tg)
				if err != nil {
					c.plugin.logger.Error(fmt.Sprintf("execute poll task error: %v", err))
				}
				c.lastReq = time.Now()
			}
		}
	}()
	return nil
}

func isReadable(rw string) bool {
	return strings.Contains(strings.ToUpper(rw), "R")
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

func (c *connector) executePollTask(tg *taskGroup) error {
	c.ensureInterval()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	err := c.client.Open()
	if err != nil {
		offlineDeviceNames := make(map[string]bool)
		//devSN去重
		for _, point := range tg.points {
			if _, ok := offlineDeviceNames[point.DeviceSn]; !ok {
				offlineDeviceNames[point.DeviceSn] = true
			}
		}
		for devSn, _ := range offlineDeviceNames {
			err := helper.DeviceShadow.MayBeOffline(devSn)
			if err != nil {
				c.plugin.logger.Warn(fmt.Sprintf("set offline device error: %v", err))
			}
		}
		return fmt.Errorf("open modbus client error: %s", err.Error())
	}
	defer func() {
		err = c.client.Close()
		if err != nil {
			c.plugin.logger.Error(fmt.Sprintf("close plugin error: %v", err))
		}
	}()
	values, err := c.read(tg.slaveId, string(tg.registerType), tg.address, tg.length)
	if err != nil {
		return fmt.Errorf("read %+v error: %s", tg, err.Error())
	}
	devicePointsMap := make(map[string][]PointValue)
	// 转化数据并上报
	for _, point := range tg.points {
		deviceSn := point.DeviceSn
		devicePoints, ok := devicePointsMap[deviceSn]
		if !ok {
			devicePoints = make([]PointValue, 0)
		}
		pointValue := PointValue{
			Name: point.Name,
		}
		var value interface{}
		start := point.Address - tg.address
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
		if point.Scale == "" {
			pointValue.Value = value
		} else {
			scale, err := strconv.ParseFloat(point.Scale, 64)
			if err != nil {
				c.plugin.logger.Error(fmt.Sprintf("parse scale error: %s -> %s", point.Name, point.Scale))
				continue
			}
			res, err := multiplyWithFloat64(value, scale)
			if err != nil {
				c.plugin.logger.Error(fmt.Sprintf("multiply error: %s -> %v * %s", point.Name, value, point.Scale))
				continue
			}
			pointValue.Value = res
		}
		devicePoints = append(devicePoints, pointValue)
		devicePointsMap[deviceSn] = devicePoints
	}
	_, err = callback.OnReceiveHandler(c.plugin, devicePointsMap)
	if err != nil {
		return fmt.Errorf("upload data error: %v", err)
	}
	return nil
}

func multiplyWithFloat64(value interface{}, scale float64) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v * scale, nil
	case int16:
		return float64(v) * scale, nil
	case uint16:
		return float64(v) * scale, nil
	case uint32:
		return float64(v) * scale, nil
	case int32:
		return float64(v) * scale, nil
	case int64:
		return float64(v) * scale, nil
	case uint64:
		return float64(v) * scale, nil
	case float32:
		return float64(v) * scale, nil
	default:
		return 0, fmt.Errorf("cannot multiply %T with float64", value)
	}
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

func (c *connector) sendWriteCommand(data interface{}) error {
	cmd, ok := data.(command)
	if !ok {
		return fmt.Errorf("convert to command error: %v", data)
	}
	pointMap, ok := c.pointMap[cmd.device]
	if !ok {
		return fmt.Errorf("unable to find device point map: %v", cmd.device)
	}
	pc, ok := pointMap[cmd.point]
	if !ok {
		return fmt.Errorf("unable to get point configuration: %v->%v", cmd.device, cmd.point)
	}
	if !isWritable(pc.ReadWrite) {
		return fmt.Errorf("unable to write read only point： %v", pc)
	}
	var values []uint16
	value := cmd.value
	switch pc.RegisterType {
	case Coil: // 线圈固定长度1
		i, err := helper.Conv2Int64(value)
		if err != nil {
			return fmt.Errorf("convert cmd  error: %v", err)
		}
		values = []uint16{uint16(i & 1)}
	case HoldingRegister:
		valueStr := fmt.Sprintf("%v", value)
		if pc.Scale != "" {
			v, err := divideStrings(valueStr, pc.Scale)
			if err != nil {
				return fmt.Errorf("process value scale error: %v", err)
			}
			valueStr = v
		}
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
				uint16s, err := c.read(pc.SlaveId, string(pc.RegisterType), pc.Address, pc.Quantity)
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
				uint16s, err := c.read(pc.SlaveId, string(pc.RegisterType), pc.Address, pc.Quantity)
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
		if err = c.write(pc.SlaveId, pc.RegisterType, pc.Address, values); err == nil {
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

type pointConfig struct {
	DeviceSn     string
	Name         string
	ReadWrite    string
	SlaveId      uint8
	RegisterType primaryTable `json:"primaryTable"`
	Address      uint16
	Quantity     uint16 `json:"quantity"`
	Bit          int    `json:"bit"`
	BitLen       int    `json:"bitLen"`
	RawType      string `json:"rawType"`
	ByteSwap     bool   `json:"byteSwap"`
	WordSwap     bool   `json:"wordSwap"`
	Scale        string `json:"scale"`
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
