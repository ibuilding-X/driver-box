package tcpserver

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/core/contracts"
	"github.com/ibuilding-x/driver-box/core/helper"
	"github.com/ibuilding-x/driver-box/internal/driver/common"
	lua "github.com/yuin/gopher-lua"
	"sync"
)

// Adapter 协议适配器
type adapter struct {
	scriptDir string // 脚本目录名称
	ls        *lua.LState
	lock      *sync.Mutex
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
	a.lock.Lock()
	defer a.lock.Unlock()
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
