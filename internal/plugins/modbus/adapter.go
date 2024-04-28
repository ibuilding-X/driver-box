package modbus

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Decode 解码数据
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	readValue, ok := raw.(plugin.PointReadValue)
	if !ok {
		return nil, fmt.Errorf("unexpected raw: %v", raw)
	}

	if c.ScriptEnable {
		resBytes, err := json.Marshal(readValue)
		if err != nil {
			return nil, fmt.Errorf("marshal result [%v] error: %v", res, err)
		}
		return helper.CallLuaConverter(c.Ls, "decode", string(resBytes))
	} else {
		res = append(res, plugin.DeviceData{
			SN: readValue.SN,
			Values: []plugin.PointData{{
				PointName: readValue.PointName,
				Value:     readValue.Value,
			}},
		})
	}
	return
}

// Encode 编码数据
func (c *connector) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	if mode == plugin.WriteMode {
		writeValues, err := c.batchWriteEncode(deviceSn, values)
		return command{
			Mode:  plugin.WriteMode,
			Value: writeValues,
		}, err
	}
	return nil, fmt.Errorf("unsupported mode %v", plugin.ReadMode)
}

func (c *connector) batchWriteEncode(deviceSn string, points []plugin.PointData) ([]*writeValue, error) {
	values := make([]*writeValue, 0)
	for _, p := range points {
		wv, err := c.getWriteValue(deviceSn, p)
		if err != nil {
			return values, err
		}
		values = append(values, &wv)
	}
	//按照address排序
	sort.Slice(values, func(i, j int) bool {
		return values[i].Address < values[j].Address
	})

	mergedValues := make([]*writeValue, 0)
	var preValue *writeValue
	for _, v := range values {
		//仅保持寄存器支持批量
		if v.RegisterType != HoldingRegister {
			mergedValues = append(mergedValues, v)
			continue
		}
		if preValue == nil {
			preValue = v
			mergedValues = append(mergedValues, v)
			continue
		}

		//批量下发必须为连续地址
		if int(preValue.Address)+len(preValue.Value) != int(v.Address) {
			preValue = v
			mergedValues = append(mergedValues, v)
			continue
		}
		//超过批量写支持的字节长度范围
		batchLen := len(preValue.Value) + len(v.Value)
		if batchLen > c.config.BatchWriteLen {
			preValue = v
			mergedValues = append(mergedValues, v)
			continue
		}
		//合并数据
		bytes := make([]uint16, batchLen)
		copy(bytes, preValue.Value)
		copy(bytes[len(preValue.Value):], v.Value)
		preValue.Value = bytes
	}
	return mergedValues, nil
}
func (c *connector) getWriteValue(deviceSn string, pointData plugin.PointData) (writeValue, error) {
	value := pointData.Value
	d, ok := helper.CoreCache.GetDevice(deviceSn)
	if !ok {
		return writeValue{}, errors.New("device not found")
	}
	unitId, err := getUnitId(d.Properties)
	if err != nil {
		return writeValue{}, err
	}
	p, ok := helper.CoreCache.GetPointByDevice(deviceSn, pointData.PointName)
	if !ok {
		return writeValue{}, errors.New("point not found")
	}

	ext, err := convToPointExtend(p.Extends)
	if err != nil {
		helper.Logger.Error("error modbus point config", zap.String("deviceSn", deviceSn), zap.Any("point", pointData.PointName), zap.Error(err))
		return writeValue{}, err
	}
	var values []uint16
	switch ext.RegisterType {
	case Coil: // 线圈固定长度1
		i, err := helper.Conv2Int64(value)
		if err != nil {
			return writeValue{}, err
		}
		values = []uint16{uint16(i & 1)}
	case HoldingRegister:
		valueStr := fmt.Sprintf("%v", value)
		switch strings.ToUpper(ext.RawType) {
		case strings.ToUpper(common.ValueTypeUint16):
			v, err := strconv.ParseUint(valueStr, 10, 16)
			if err != nil {
				return writeValue{}, fmt.Errorf("convert value %v to uint16 error: %v", value, err)
			}
			// TODO: 位写
			if ext.BitLen > 0 {
				if v > (1<<ext.BitLen - 1) {
					return writeValue{}, fmt.Errorf("too large value %v to set in %d bits", v, ext.BitLen)
				}
				uint16s, err := c.read(unitId, string(ext.RegisterType), ext.Address, ext.Quantity)
				uint16Val := uint16s[0]
				if ext.ByteSwap {
					uint16Val = (uint16Val << 8) | (uint16Val >> 8)
				}
				if err != nil {
					return writeValue{}, fmt.Errorf("read original register error: %v", err)
				}
				intoUint16 := mergeBitsIntoUint16(int(v), ext.Bit, ext.BitLen, uint16Val)
				if ext.ByteSwap {
					intoUint16 = (intoUint16 << 8) | (intoUint16 >> 8)
				}
				values = []uint16{intoUint16}
				break
			}
			out := make([]byte, 2)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint16(out, uint16(v))
			} else {
				binary.BigEndian.PutUint16(out, uint16(v))
			}
			values = []uint16{binary.BigEndian.Uint16(out)}
		case strings.ToUpper(common.ValueTypeInt16):
			v, err := strconv.ParseInt(valueStr, 10, 16)
			if err != nil {
				return writeValue{}, fmt.Errorf("convert value %v to int16 error: %v", value, err)
			}
			if ext.BitLen > 0 {
				if v > (1<<ext.BitLen - 1) {
					return writeValue{}, fmt.Errorf("too large value %v to set in %d bits", v, ext.BitLen)
				} else if v < 0 {
					return writeValue{}, fmt.Errorf("negative value %v not allowed to set in bits", v)
				}
				uint16s, err := c.read(unitId, string(ext.RegisterType), ext.Address, ext.Quantity)
				uint16Val := uint16s[0]
				if ext.ByteSwap {
					uint16Val = (uint16Val << 8) | (uint16Val >> 8)
				}
				if err != nil {
					return writeValue{}, fmt.Errorf("read original register error: %v", err)
				}
				intoUint16 := mergeBitsIntoUint16(int(v), ext.Bit, ext.BitLen, uint16Val)
				if ext.ByteSwap {
					intoUint16 = (intoUint16 << 8) | (intoUint16 >> 8)
				}
				values = []uint16{intoUint16}
				break
			}
			out := make([]byte, 2)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint16(out, uint16(v))
			} else {
				binary.BigEndian.PutUint16(out, uint16(v))
			}
			values = []uint16{binary.BigEndian.Uint16(out)}
		case strings.ToUpper(common.ValueTypeUint32):
			v, err := strconv.ParseUint(valueStr, 10, 32)
			if err != nil {
				return writeValue{}, fmt.Errorf("convert value %v to uint32 error: %v", value, err)
			}
			out := make([]byte, 4)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint32(out, uint32(v))
			} else {
				binary.BigEndian.PutUint32(out, uint32(v))
			}
			if ext.WordSwap {
				out[0], out[1], out[2], out[3] =
					out[2], out[3], out[0], out[1]
			}
			values = []uint16{binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]})}
		case strings.ToUpper(common.ValueTypeInt32):
			v, err := strconv.ParseInt(valueStr, 10, 32)
			if err != nil {
				return writeValue{}, fmt.Errorf("convert value %v to int32 error: %v", value, err)
			}
			out := make([]byte, 4)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint32(out, uint32(v))
			} else {
				binary.BigEndian.PutUint32(out, uint32(v))
			}
			if ext.WordSwap {
				out[0], out[1], out[2], out[3] =
					out[2], out[3], out[0], out[1]
			}
			values = []uint16{binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]})}
		case strings.ToUpper(common.ValueTypeFloat32):
			v, err := strconv.ParseFloat(valueStr, 32)
			if err != nil {
				return writeValue{}, fmt.Errorf("convert value %v to float32 error: %v", value, err)
			}
			v32 := float32(v)
			bits := math.Float32bits(v32)
			out := make([]byte, 4)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint32(out, bits)
			} else {
				binary.BigEndian.PutUint32(out, bits)
			}
			if ext.WordSwap {
				out[0], out[1], out[2], out[3] =
					out[2], out[3], out[0], out[1]
			}
			values = []uint16{binary.BigEndian.Uint16([]byte{out[2], out[3]}),
				binary.BigEndian.Uint16([]byte{out[0], out[1]})}
		case strings.ToUpper(common.ValueTypeUint64):
			v, err := strconv.ParseUint(valueStr, 10, 64)
			if err != nil {
				return writeValue{}, fmt.Errorf("convert value %v to uint64 error: %v", value, err)
			}
			out := make([]byte, 8)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint64(out, v)
			} else {
				binary.BigEndian.PutUint64(out, v)
			}
			if ext.WordSwap {
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
				return writeValue{}, fmt.Errorf("convert value %v to int64 error: %v", value, err)
			}
			out := make([]byte, 8)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint64(out, uint64(v))
			} else {
				binary.BigEndian.PutUint64(out, uint64(v))
			}
			if ext.WordSwap {
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
				return writeValue{}, fmt.Errorf("convert value %v to float64 error: %v", value, err)
			}
			out := make([]byte, 8)
			if ext.ByteSwap {
				binary.LittleEndian.PutUint64(out, math.Float64bits(v))
			} else {
				binary.BigEndian.PutUint64(out, math.Float64bits(v))
			}
			if ext.WordSwap {
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
			if len(valueBytes) > int(ext.Quantity*2) {
				return writeValue{}, fmt.Errorf("too long string [%v] to set in %d registers", valueStr, ext.Quantity)
			}
			if ext.ByteSwap {
				for i := 0; i < len(valueBytes); i += 2 {
					if i+1 < len(valueBytes) {
						valueBytes[i], valueBytes[i+1] = valueBytes[i+1], valueBytes[i]
					}
				}
			}
			if ext.WordSwap {
				for i := 0; i < len(valueBytes); i += 4 {
					if i+3 < len(valueBytes) {
						valueBytes[i], valueBytes[i+1], valueBytes[i+2], valueBytes[i+3] =
							valueBytes[i+2], valueBytes[i+3], valueBytes[i], valueBytes[i+1]
					}
				}
			}
			values = make([]uint16, ext.Quantity)
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
			return writeValue{}, fmt.Errorf("unsupported raw type: %v", ext)
		}
	default:
		return writeValue{}, fmt.Errorf("unsupported write register type: %v", ext)
	}
	return writeValue{
		unitID:       unitId,
		RegisterType: ext.RegisterType,
		Address:      ext.Address,
		Value:        values,
	}, nil
}
