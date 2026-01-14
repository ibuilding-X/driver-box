package luautil

import (
	"github.com/ibuilding-x/driver-box/internal/core/cache"
	"github.com/ibuilding-x/driver-box/internal/logger"
	"github.com/ibuilding-x/driver-box/internal/shadow"
	"github.com/ibuilding-x/driver-box/pkg/config"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
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
		//"cache_set": lm.SetCache,
		//"cache_get": lm.GetCache,
		"shadow": lm.getDeviceShadow,
		//"device_list": lm.getDeviceShadow,
		"getDevice":     lm.getDevice,
		"setDeviceProp": lm.setDeviceProp,
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
	device, ok := shadow.DeviceShadow.GetDevice(deviceId)
	points := L.NewTable()
	defer L.Push(points)
	if !ok {
		return 1
	}
	for _, point := range device.Points {
		p, ok := cache.Instance.GetPointByDevice(deviceId, point.Name)
		if !ok {
			logger.Logger.Error("could not get point", zap.String("deviceId", deviceId), zap.String("pointName", point.Name))
			continue
		}
		switch p.ValueType() {
		case config.ValueType_String:
			v, e := convutil.String(point.Value)
			if e == nil {
				points.RawSetString(point.Name, lua.LString(v))
			} else {
				logger.Logger.Error("could not conv2 string", zap.String("deviceId", deviceId), zap.Any("point", point), zap.Error(e))
			}
		case config.ValueType_Int:
			v, e := convutil.Int64(point.Value)
			if e == nil {
				points.RawSetString(point.Name, lua.LNumber(v))
			} else {
				logger.Logger.Error("could not conv2 int", zap.String("deviceId", deviceId), zap.Any("point", point), zap.Error(e))
			}
		case config.ValueType_Float:
			v, e := convutil.Float64(point.Value)
			if e == nil {
				points.RawSetString(point.Name, lua.LNumber(v))
			} else {
				logger.Logger.Error("could not conv2 float", zap.String("deviceId", deviceId), zap.Any("point", point), zap.Error(e))
			}
		}
	}
	return 1
}

// getDevice  获取指定ID的影子信息
func (lm *LuaModule) getDevice(L *lua.LState) int {
	deviceId := L.ToString(1)
	device, ok := cache.Instance.GetDevice(deviceId)
	deviceTable := L.NewTable()
	if !ok {
		return 0
	}
	defer L.Push(deviceTable)
	deviceTable.RawSetString("id", lua.LString(device.ID))
	deviceTable.RawSetString("modelName", lua.LString(device.ModelName))
	deviceTable.RawSetString("driverKey", lua.LString(device.DriverKey))
	deviceTable.RawSetString("connectionKey", lua.LString(device.ConnectionKey))
	properties := L.NewTable()
	for k, v := range device.Properties {
		properties.RawSetString(k, lua.LString(v))
	}
	deviceTable.RawSetString("properties", properties)
	return 1
}

/**
 * setDeviceProps 设置设备属性
 */
func (lm *LuaModule) setDeviceProp(L *lua.LState) int {
	deviceId := L.ToString(1)
	propName := L.ToString(2)
	propValue := L.ToString(3)
	deviceTable := L.NewTable()
	defer L.Push(deviceTable)
	cache.Instance.UpdateDeviceProperty(deviceId, propName, propValue)
	return 1
}

// GetCache 获取缓存
//func (lm *LuaModule) GetCache(L *lua.LState) int {
//	// 获取 lua 参数
//	key := L.ToString(1)
//	if key != "" {
//		if value, ok := helper.PluginCacheMap.Load(key); ok {
//			v := fmt.Sprintf("%v", value)
//			// 结果返回
//			L.Push(lua.LString(v))
//			return 1
//		}
//	}
//
//	L.Push(lua.LNil)
//	return 1
//}
//
//// SetCache 设置缓存
//func (lm *LuaModule) SetCache(L *lua.LState) int {
//	// 获取参数
//	key := L.ToString(1)
//	value := L.ToString(2)
//	helper.PluginCacheMap.Store(key, value)
//	return 0
//}
