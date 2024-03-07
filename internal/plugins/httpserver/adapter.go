package httpserver

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

// protoData 协议数据，框架重组交由动态脚本解析
type protoData struct {
	Path   string `json:"path"`   // 请求路径
	Method string `json:"method"` // 请求方法
	Body   string `json:"body"`   // 请求 body
	// todo 后续待扩充
}

// ToJSON 协议数据转 json 字符串
func (pd protoData) ToJSON() string {
	b, _ := json.Marshal(pd)
	return string(b)
}

// Encode 编码数据，无需实现
func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, values plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Decode 解码数据，调用动态脚本解析
func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
