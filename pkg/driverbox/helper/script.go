package helper

import (
	"github.com/ibuilding-x/driver-box/internal/lua"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/common"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	glua "github.com/yuin/gopher-lua"
	"path/filepath"
)

// CallLuaConverter 调用 Lua 脚本转换器
func CallLuaConverter(L *glua.LState, method string, raw interface{}) ([]plugin.DeviceData, error) {
	return lua.CallLuaConverter(L, method, raw)
}

// 执行指定lua方法
func CallLuaMethod(L *glua.LState, method string, args ...glua.LValue) (string, error) {
	return lua.CallLuaMethod(L, method, args...)
}

// CallLuaEncodeConverter 调用 Lua 脚本编码转换器
func CallLuaEncodeConverter(L *glua.LState, deviceSn string, raw interface{}) (string, error) {
	return lua.CallLuaEncodeConverter(L, deviceSn, raw)
}

// 关闭Lua虚拟机
func Close(L *glua.LState) {
	lua.Close(L)
}

// scriptExists 判断lua脚本是否存在
func ScriptExists(dir string) bool {
	return common.FileExists(filepath.Join(EnvConfig.ConfigPath, dir, common.LuaScriptName))
}
