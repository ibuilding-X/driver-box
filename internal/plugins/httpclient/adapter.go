package httpclient

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
)

// Adapter 协议适配器
type adapter struct {
	scriptDir string      // 脚本目录名称
	ls        *lua.LState // lua 虚拟机
}

// transportationData 通讯数据（编码返回、解码输入数据格式）
type transportationData struct {
	Mode     string           `json:"mode"`   // 读取模式
	Values   plugin.PointData `json:"values"` // 写入值，仅当 write 模式时使用
	Protocol connectorConfig  `json:"protocol"`
}

// ToJSON 协议数据转 json 字符串
func (td transportationData) ToJSON() string {
	b, _ := json.Marshal(td)
	return string(b)
}

// Encode 编码数据
func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	if len(values) != 1 {
		return nil, common.NotSupportEncode
	}
	value := values[0]
	data := transportationData{
		Mode:     string(mode),
		Values:   value,
		Protocol: connectorConfig{},
	}
	return helper.CallLuaEncodeConverter(a.ls, deviceSn, data.ToJSON())
}

// Decode 解码数据，调用动态脚本解析
func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
