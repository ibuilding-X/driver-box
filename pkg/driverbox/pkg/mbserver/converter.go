package mbserver

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/util/gconv"
	"math"
	"strconv"
)

type converter struct {
}

func (c *converter) Uint16Slice(v interface{}, valueType int) ([]uint16, error) {
	switch valueType {
	case ValueTypeBool:
		if gconv.Bool(v) {
			return []uint16{1}, nil
		}
		return []uint16{0}, nil
	case ValueTypeUint8:
		return []uint16{uint16(gconv.Uint8(v))}, nil
	case ValueTypeInt8:
		return []uint16{uint16(gconv.Int8(v))}, nil
	case ValueTypeUint16:
		return []uint16{gconv.Uint16(v)}, nil
	case ValueTypeInt16:
		return []uint16{uint16(gconv.Int16(v))}, nil
	case ValueTypeUint32:
		value := gconv.Uint32(v)
		return []uint16{uint16(value >> 16), uint16(value & 0xFFFF)}, nil
	case ValueTypeInt32:
		value := gconv.Int32(v)
		return []uint16{uint16(value >> 16), uint16(value & 0xFFFF)}, nil
	case ValueTypeUint64:
		value := gconv.Uint64(v)
		return []uint16{uint16(value >> 48), uint16(value >> 32 & 0xFFFF), uint16(value >> 16 & 0xFFFF), uint16(value & 0xFFFF)}, nil
	case ValueTypeInt64:
		value := gconv.Int64(v)
		return []uint16{uint16(value >> 48), uint16(value >> 32 & 0xFFFF), uint16(value >> 16 & 0xFFFF), uint16(value & 0xFFFF)}, nil
	case ValueTypeUint:
		value := gconv.Uint(v)
		switch strconv.IntSize {
		case 64:
			return []uint16{uint16(value >> 48), uint16(value >> 32 & 0xFFFF), uint16(value >> 16 & 0xFFFF), uint16(value & 0xFFFF)}, nil
		default:
			return []uint16{uint16(value >> 16 & 0xFFFF), uint16(value & 0xFFFF)}, nil
		}
	case ValueTypeInt:
		value := gconv.Int(v)
		switch strconv.IntSize {
		case 64:
			return []uint16{uint16(value >> 48), uint16(value >> 32 & 0xFFFF), uint16(value >> 16 & 0xFFFF), uint16(value & 0xFFFF)}, nil
		default:
			return []uint16{uint16(value >> 16 & 0xFFFF), uint16(value & 0xFFFF)}, nil
		}
	case ValueTypeFloat32:
		value := gconv.Float32(v)
		u32 := math.Float32bits(value)
		return []uint16{uint16(u32 >> 16), uint16(u32 & 0xFFFF)}, nil
	case ValueTypeFloat64:
		value := gconv.Float64(v)
		u64 := math.Float64bits(value)
		return []uint16{uint16(u64 >> 48), uint16(u64 >> 32 & 0xFFFF), uint16(u64 >> 16 & 0xFFFF), uint16(u64 & 0xFFFF)}, nil
	default:
		return nil, fmt.Errorf("unsupport value type [%d]", valueType)
	}
}

func (c *converter) ConvUint16Slice(uint16Slice []uint16, valueType int) (interface{}, error) {
	switch valueType {
	case ValueTypeBool:
		if len(uint16Slice) != 1 {
			return nil, errors.New("invalid uint16 slice length")
		}
		if uint16Slice[0] == 0 {
			return false, nil
		}
		return true, nil
	case ValueTypeUint8:
		if len(uint16Slice) != 1 {
			return nil, errors.New("invalid uint16 slice length")
		}
		return uint8(uint16Slice[0] & 0xFF), nil
	case ValueTypeInt8:
		if len(uint16Slice) != 1 {
			return nil, errors.New("invalid uint16 slice length")
		}
		return int8(uint16Slice[0] & 0xFF), nil
	case ValueTypeUint16:
		if len(uint16Slice) != 1 {
			return nil, errors.New("invalid uint16 slice length")
		}
		return uint16Slice[0], nil
	case ValueTypeInt16:
		if len(uint16Slice) != 1 {
			return nil, errors.New("invalid uint16 slice length")
		}
		return int16(uint16Slice[0]), nil
	case ValueTypeUint32:
		if len(uint16Slice) != 2 {
			return nil, errors.New("invalid uint16 slice length")
		}
		var u32 uint32
		u32 = uint32(uint16Slice[0])<<16 | uint32(uint16Slice[1])
		return u32, nil
	case ValueTypeInt32:
		if len(uint16Slice) != 2 {
			return nil, errors.New("invalid uint16 slice length")
		}
		var i32 int32
		i32 = int32(uint16Slice[0])<<16 | int32(uint16Slice[1])
		return i32, nil
	case ValueTypeUint64:
		if len(uint16Slice) != 4 {
			return nil, errors.New("invalid uint16 slice length")
		}
		var u64 uint64
		u64 = uint64(uint16Slice[0])<<48 | uint64(uint16Slice[1])<<32 | uint64(uint16Slice[2])<<16 | uint64(uint16Slice[3])
		return u64, nil
	case ValueTypeInt64:
		if len(uint16Slice) != 4 {
			return nil, errors.New("invalid uint16 slice length")
		}
		var i64 int64
		i64 = int64(uint16Slice[0])<<48 | int64(uint16Slice[1])<<32 | int64(uint16Slice[2])<<16 | int64(uint16Slice[3])
		return i64, nil
	case ValueTypeUint:
		switch strconv.IntSize {
		case 64:
			if len(uint16Slice) != 4 {
				return nil, errors.New("invalid uint16 slice length")
			}
			var u64 uint64
			u64 = uint64(uint16Slice[0])<<16 | uint64(uint16Slice[1])
			return uint(u64), nil
		default:
			if len(uint16Slice) != 2 {
				return nil, errors.New("invalid uint16 slice length")
			}
			var u32 uint32
			u32 = uint32(uint16Slice[0])<<16 | uint32(uint16Slice[1])
			return uint(u32), nil
		}
	case ValueTypeInt:
		switch strconv.IntSize {
		case 64:
			if len(uint16Slice) != 4 {
				return nil, errors.New("invalid uint16 slice length")
			}
			var i64 int64
			i64 = int64(uint16Slice[0])<<16 | int64(uint16Slice[1])
			return int(i64), nil
		default:
			if len(uint16Slice) != 2 {
				return nil, errors.New("invalid uint16 slice length")
			}
			var i32 int32
			i32 = int32(uint16Slice[0])<<16 | int32(uint16Slice[1])
			return int(i32), nil
		}
	case ValueTypeFloat32:
		if len(uint16Slice) != 2 {
			return nil, errors.New("invalid uint16 slice length")
		}
		var u32 uint32
		u32 = uint32(uint16Slice[0])<<16 | uint32(uint16Slice[1])

		var f32 float32
		f32 = math.Float32frombits(u32)
		return f32, nil
	case ValueTypeFloat64:
		if len(uint16Slice) != 4 {
			return nil, errors.New("invalid uint16 slice length")
		}
		var u64 uint64
		u64 = uint64(uint16Slice[0])<<48 | uint64(uint16Slice[1])<<32 | uint64(uint16Slice[2])<<16 | uint64(uint16Slice[3])

		var f64 float64
		f64 = math.Float64frombits(u64)
		return f64, nil
	default:
		return nil, fmt.Errorf("unsupported value type: %d", valueType)
	}
}
