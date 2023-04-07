package tcpserver

import (
	"driver-box/core/contracts"
	"driver-box/core/helper"
	"driver-box/driver/common"
	"encoding/json"
	lua "github.com/yuin/gopher-lua"
)

// Adapter 协议适配器
type adapter struct {
	scriptDir string // 脚本目录名称
	ls        *lua.LState
}

// protoData 协议数据
type protoData struct {
	Raw string `json:"raw"`
}

// ToJSON 协议数据转 json 字符串
func (pd protoData) ToJSON() string {
	b, _ := json.Marshal(pd)
	return string(b)
}

// Encode 编码
// 暂无实现
func (a *adapter) Encode(deviceName string, mode contracts.EncodeMode, value contracts.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Decode 解码
func (a *adapter) Decode(raw interface{}) (res []contracts.DeviceData, err error) {
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
