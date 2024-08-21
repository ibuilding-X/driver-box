package lua

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/logger"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

var LuaModuleInstance *LuaModule

// LuaModule go 实现 lua 模块，供脚本内使用
type LuaModule struct{}

// Loader 模块预加载
func (lm *LuaModule) Loader(L *lua.LState) int {
	// 模块方法定义
	mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"cache_set": lm.SetCache,
		"cache_get": lm.GetCache,
		"shadow":    lm.getDeviceShadow,
		//"device_list": lm.getDeviceShadow,
		//"device_get":  lm.getDeviceShadow,
		//"writeToMsgBus": lm.WriteToMsgBus,
	})
	L.Push(mod)

	return 1
}

func (lm *LuaModule) getDeviceList(L *lua.LState) int {
	//deviceId := L.ToString(1)
	//pointName := L.ToString(2)
	//devices := helper.DeviceShadow.GetDevices()
	//L.Push(lua.LTable{}(devices))
	return 0
}

// getDeviceShadow 获取指定ID的设备影子
func (lm *LuaModule) getDeviceShadow(L *lua.LState) int {
	deviceId := L.ToString(1)
	device, ok := helper.DeviceShadow.GetDevice(deviceId)
	points := L.NewTable()
	defer L.Push(points)
	if !ok {
		return 1
	}
	for _, point := range device.Points {
		p, ok := helper.CoreCache.GetPointByDevice(deviceId, point.Name)
		if !ok {
			logger.Logger.Error("could not get point", zap.String("deviceId", deviceId), zap.String("pointName", point.Name))
			continue
		}
		switch p.ValueType {
		case config.ValueType_String:
			v, e := helper.Conv2String(point.Value)
			if e == nil {
				points.RawSetString(point.Name, lua.LString(v))
			} else {
				logger.Logger.Error("could not conv2 string", zap.String("deviceId", deviceId), zap.Any("point", point), zap.Error(e))
			}
		case config.ValueType_Int:
			v, e := helper.Conv2Int64(point.Value)
			if e == nil {
				points.RawSetString(point.Name, lua.LNumber(v))
			} else {
				logger.Logger.Error("could not conv2 int", zap.String("deviceId", deviceId), zap.Any("point", point), zap.Error(e))
			}
		case config.ValueType_Float:
			v, e := helper.Conv2Float64(point.Value)
			if e == nil {
				points.RawSetString(point.Name, lua.LNumber(v))
			} else {
				logger.Logger.Error("could not conv2 float", zap.String("deviceId", deviceId), zap.Any("point", point), zap.Error(e))
			}
		}
	}
	return 1
}

// GetCache 获取缓存
func (lm *LuaModule) GetCache(L *lua.LState) int {
	// 获取 lua 参数
	key := L.ToString(1)
	if key != "" {
		if value, ok := helper.PluginCacheMap.Load(key); ok {
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
	helper.PluginCacheMap.Store(key, value)
	return 0
}
