package mbslave

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// convUint16s 转换数据为 []uint16
// dstType 目前支持的类型有：uint16、float32
func convUint16s(dstType string, v interface{}) ([]uint16, error) {
	switch dstType {
	case "uint16":
		if u16, err := convUint16(v); err != nil {
			return nil, err
		} else {
			return []uint16{u16}, nil
		}
	case "float32":
		if f32, err := convFloat32(v); err != nil {
			return nil, err
		} else {
			u32 := math.Float32bits(f32)
			return []uint16{uint16(u32 >> 16), uint16(u32 & 0xFFFF)}, nil
		}
	default:
		return nil, fmt.Errorf("unsupported type %s", dstType)
	}
}

// convUint16 转换数据为 uint16
func convUint16(v interface{}) (uint16, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	case reflect.String:
		f64, err := strconv.ParseFloat(rv.String(), 64)
		if err != nil {
			return 0, err
		}
		return uint16(f64), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uint16(rv.Uint()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint16(rv.Int()), nil
	case reflect.Float32, reflect.Float64:
		return uint16(rv.Float()), nil
	default:
		return 0, fmt.Errorf("converting %s to uint16 is not supported", rv.Kind())
	}
}

// convFloat32 转换数据为 float32
func convFloat32(v interface{}) (float32, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		if rv.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	case reflect.String:
		f64, err := strconv.ParseFloat(rv.String(), 64)
		if err != nil {
			return 0, err
		}
		return float32(f64), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float32(rv.Uint()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float32(rv.Int()), nil
	case reflect.Float32, reflect.Float64:
		return float32(rv.Float()), nil
	default:
		return 0, fmt.Errorf("converting %s to float32 is not supported", rv.Kind())
	}
}
