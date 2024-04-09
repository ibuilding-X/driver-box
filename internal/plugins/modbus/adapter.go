package modbus

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
)

// Adapter 协议适配器
type adapter struct {
	scriptDir string // 脚本目录名称
	ls        *lua.LState
}

// Decode 解码数据
func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	deviceData, ok := raw.(map[string][]PointValue)
	if !ok {
		return nil, fmt.Errorf("unexpected raw: %v", raw)
	}
	for deviceSn, pointValues := range deviceData {
		dd := plugin.DeviceData{
			SN:     deviceSn,
			Values: make([]plugin.PointData, 0),
		}
		for _, pointValue := range pointValues {
			dd.Values = append(dd.Values, plugin.PointData{
				PointName: pointValue.Name,
				Value:     pointValue.Value,
			})
		}
		res = append(res, dd)
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("marshal result [%v] error: %v", res, err)
	}
	if a.scriptExists() {
		return helper.CallLuaConverter(a.ls, "decode", string(resBytes))
	}
	return
}

// Encode 编码数据
func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, value plugin.PointData) (res interface{}, err error) {
	if mode == plugin.ReadMode {
		return nil, fmt.Errorf("unsupported mode %v", plugin.ReadMode)
	}
	res = command{
		device: deviceSn,
		point:  value.PointName,
		value:  value.Value,
	}
	return res, nil
}

type command struct {
	device string
	point  string
	value  interface{}
}

// scriptExists 判断lua脚本是否存在
func (a *adapter) scriptExists() bool {
	scriptPath := filepath.Join(helper.EnvConfig.ConfigPath, a.scriptDir, common.LuaScriptName)
	_, err := os.Stat(scriptPath)
	return err == nil
}
