package modbus

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// Adapter 协议适配器
type adapter struct {
	scriptEnable bool //是否存在动态脚本
	ls           *lua.LState
}

// Decode 解码数据
func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	readValue, ok := raw.(plugin.PointReadValue)
	if !ok {
		return nil, fmt.Errorf("unexpected raw: %v", raw)
	}

	if a.scriptEnable {
		resBytes, err := json.Marshal(readValue)
		if err != nil {
			return nil, fmt.Errorf("marshal result [%v] error: %v", res, err)
		}
		return helper.CallLuaConverter(a.ls, "decode", string(resBytes))
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
func (a *adapter) BatchEncode(deviceSn string, mode plugin.EncodeMode, value []plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Encode 编码数据
func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, value plugin.PointData) (res interface{}, err error) {
	if mode == plugin.ReadMode {
		return nil, fmt.Errorf("unsupported mode %v", plugin.ReadMode)
	}
	if mode == plugin.WriteMode {
		d, ok := helper.CoreCache.GetDevice(deviceSn)
		if !ok {
			return nil, errors.New("device not found")
		}
		unitId, err := getUnitId(d.Properties)
		if err != nil {
			return nil, err
		}
		p, ok := helper.CoreCache.GetPointByDevice(deviceSn, value.PointName)
		if !ok {
			return nil, errors.New("point not found")
		}

		ext, err := convToPointExtend(p.Extends)
		if err != nil {
			helper.Logger.Error("error modbus point config", zap.String("deviceSn", deviceSn), zap.Any("point", value.PointName), zap.Error(err))
			return nil, err
		}
		return command{
			mode: plugin.WriteMode,
			value: writeValue{
				unitID:       unitId,
				RegisterType: ext.RegisterType,
				Address:      ext.Address,
				Quantity:     ext.Quantity,
				BitLen:       ext.BitLen,
				WordSwap:     ext.WordSwap,
				Bit:          ext.Bit,
				ByteSwap:     ext.ByteSwap,
				RawType:      ext.RawType,
				Value:        value.Value,
			},
		}, nil
	}
	res = command{
		value: value.Value,
		mode:  mode,
	}
	return res, nil
}
