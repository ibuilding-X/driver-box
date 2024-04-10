package modbus

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
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

// Encode 编码数据
func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, value plugin.PointData) (res interface{}, err error) {
	if mode == plugin.ReadMode {
		return nil, fmt.Errorf("unsupported mode %v", plugin.ReadMode)
	}
	res = command{
		value: value.Value,
	}
	return res, nil
}
