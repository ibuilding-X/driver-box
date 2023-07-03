package httpclient

import (
	"driver-box/core/contracts"
	"driver-box/core/helper"
	"encoding/json"
	lua "github.com/yuin/gopher-lua"
	"sync"
)

// Adapter 协议适配器
type adapter struct {
	scriptDir string      // 脚本目录名称
	ls        *lua.LState // lua 虚拟机
	lock      *sync.Mutex
}

// transportationData 通讯数据（编码返回、解码输入数据格式）
type transportationData struct {
	DeviceName string              `json:"device_name"` // 设备名称
	Mode       string              `json:"mode"`        // 读取模式
	Values     contracts.PointData `json:"values"`      // 写入值，仅当 write 模式时使用
	Protocol   connectorConfig     `json:"protocol"`
}

// ToJSON 协议数据转 json 字符串
func (td transportationData) ToJSON() string {
	b, _ := json.Marshal(td)
	return string(b)
}

// Encode 编码数据
func (a *adapter) Encode(deviceName string, mode contracts.EncodeMode, values contracts.PointData) (res interface{}, err error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	data := transportationData{
		DeviceName: "deviceName",
		Mode:       string(mode),
		Values:     values,
		Protocol:   connectorConfig{},
	}
	return helper.CallLuaEncodeConverter(a.ls, deviceName, data.ToJSON())
}

// Decode 解码数据，调用动态脚本解析
func (a *adapter) Decode(raw interface{}) (res []contracts.DeviceData, err error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
