package helper

import (
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"strconv"
)

// ConvPointType 点位类型转换
// 仅支持三种数据类型：int、float、string
func ConvPointType(value interface{}, valueType config.ValueType) (interface{}, error) {
	switch valueType {
	case config.ValueType_Int:
		return Conv2Int64(value)
	case config.ValueType_Float:
		return Conv2Float64(value)
	case config.ValueType_String:
		return Conv2String(value)
	default:
		return nil, fmt.Errorf("point value type must one of (int、float、string) ,unSupport:%v", valueType)
	}
}

// Conv2Int64 转换为 int64 类型
func Conv2Int64(value interface{}) (i int64, err error) {
	if value == nil {
		return 0, nil
	}
	switch v := value.(type) {
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, errors.New(fmt.Sprintf("%T convert to int64 error", v))
	}
}

// Conv2Float64 转换为 float64 类型
func Conv2Float64(value interface{}) (f float64, err error) {
	if value == nil {
		return 0, nil
	}
	switch v := value.(type) {
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, errors.New(fmt.Sprintf("%T convert to float64 error", v))
	}
}

// Conv2String 转换为 string 类型
func Conv2String(value interface{}) (s string, err error) {
	if value == nil {
		return "", nil
	}
	switch v := value.(type) {
	case uint8:
		return strconv.FormatInt(int64(v), 10), nil
	case uint16:
		return strconv.FormatInt(int64(v), 10), nil
	case uint32:
		return strconv.FormatInt(int64(v), 10), nil
	case uint:
		return strconv.FormatInt(int64(v), 10), nil
	case uint64:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case string:
		return v, nil
	default:
		return "", errors.New(fmt.Sprintf("%T convert to string error", v))
	}
}
