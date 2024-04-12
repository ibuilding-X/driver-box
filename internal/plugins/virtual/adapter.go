package virtual

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
)

type adapter struct {
	scriptDir string
	ls        *lua.LState
}

type transportationData struct {
	SN     string             `json:"sn"`
	Mode   plugin.EncodeMode  `json:"mode"`
	Points []plugin.PointData `json:"points"`
}

func (td transportationData) ToJson() string {
	b, _ := json.Marshal(td)
	return string(b)
}
func (a *adapter) BatchEncode(deviceSn string, mode plugin.EncodeMode, value []plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}
func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, value plugin.PointData) (res interface{}, err error) {

	data := transportationData{
		SN:     deviceSn,
		Mode:   mode,
		Points: []plugin.PointData{value},
	}

	// lua 脚本处理
	if a.scriptExists() {
		var result string
		result, err = helper.CallLuaEncodeConverter(a.ls, deviceSn, data.ToJson())
		if err != nil {
			return
		}
		if err = json.Unmarshal([]byte(result), &data); err != nil {
			return
		}
	}

	return data, nil
}

func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {

	v, _ := raw.(transportationData)
	// lua 脚本处理
	if a.scriptExists() {
		return helper.CallLuaConverter(a.ls, "decode", v.ToJson())
	}

	// 点位数据为空判断
	if len(v.Points) <= 0 {
		return
	}

	// 提取所有点位数据
	var vs []plugin.PointData
	for _, point := range v.Points {
		vs = append(vs, plugin.PointData{
			PointName: point.PointName,
			Value:     point.Value,
		})
	}

	// 汇总
	res = append(res, plugin.DeviceData{
		SN:     v.SN,
		Values: vs,
	})

	return
}

// scriptExists 判断lua脚本是否存在
func (a *adapter) scriptExists() bool {
	scriptPath := filepath.Join(helper.EnvConfig.ConfigPath, a.scriptDir, common.LuaScriptNameForVirtual)
	_, err := os.Stat(scriptPath)
	return err == nil
}
