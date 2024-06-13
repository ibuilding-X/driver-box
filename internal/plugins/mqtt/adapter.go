package mqtt

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
)

// Adapter 协议适配器
type adapter struct {
	scriptDir string // 脚本目录名称
	ls        *lua.LState
}

func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	if len(values) != 1 {
		return nil, common.NotSupportEncode
	}
	value := values[0]
	if mode == plugin.WriteMode {
		tmp, _ := json.Marshal(value)
		return helper.CallLuaEncodeConverter(a.ls, deviceSn, string(tmp))
	}
	return nil, common.NotSupportEncode
}

// Decode 解析数据
func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
