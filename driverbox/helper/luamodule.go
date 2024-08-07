package helper

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
)

var LuaModuleInstance *LuaModule

// LuaModule go 实现 lua 模块，供脚本内使用
type LuaModule struct{}

// Loader 模块预加载
func (lm *LuaModule) Loader(L *lua.LState) int {
	// 模块方法定义
	mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"setCache": lm.SetCache,
		"getCache": lm.GetCache,
		//"writeToMsgBus": lm.WriteToMsgBus,
	})
	L.Push(mod)

	return 1
}

// GetCache 获取缓存
func (lm *LuaModule) GetCache(L *lua.LState) int {
	// 获取 lua 参数
	key := L.ToString(1)
	if key != "" {
		if value, ok := PluginCacheMap.Load(key); ok {
			v := fmt.Sprintf("%v", value)
			// 结果返回
			L.Push(lua.LString(v))
			return 1
		}
	}

	L.Push(lua.LNil)
	return 1
}

// SetCache 设置缓存
func (lm *LuaModule) SetCache(L *lua.LState) int {
	// 获取参数
	key := L.ToString(1)
	value := L.ToString(2)
	PluginCacheMap.Store(key, value)
	return 0
}

// WriteToMsgBus 上报设备数据
// deviceSn：设备SN，例如：sensor_1
// points：点位数据，例如（此时以json格式进行说明，lua实际入参格式为 table 类型）：{"onOff":1, "voc": 23}
//func (lm *LuaModule) WriteToMsgBus(L *lua.LState) int {
//	deviceId := L.ToString(1) // 设备名称
//	points := L.ToTable(2)    // 点位值
//
//	deviceData := plugin.DeviceData{
//		ID: deviceId,
//	}
//	var pd []plugin.PointData
//	// 循环点位数据
//	points.ForEach(func(key lua.LValue, value lua.LValue) {
//		pd = append(pd, plugin.PointData{
//			PointName: key.String(),
//			Value:     value.String(),
//		})
//	})
//	deviceData.Values = pd
//
//	PointCacheFilter(&deviceData)
//
//	// 发送数据
//	if len(deviceData.Values) > 0 {
//		for _, export := range export.Exports {
//			if export.IsReady() {
//				export.ExportTo(deviceData)
//			}
//		}
//	}
//	//WriteToMessageBus(deviceData)
//
//	return 0
//}
