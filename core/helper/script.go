package helper

import (
	"driver-box/core/contracts"
	"driver-box/driver/common"
	"encoding/json"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
)

// InitLuaVM 编译 lua 脚本
func InitLuaVM(scriptDir string) (*lua.LState, error) {
	ls := lua.NewState(lua.Options{
		RegistryMaxSize: 128,
	})
	// 文件路径
	filePath := filepath.Join(common.CoreConfigPath, scriptDir, common.LuaScriptName)
	if FileExists(filePath) {
		// 脚本解析
		err := ls.DoFile(filePath)
		if err != nil {
			return nil, err
		}
		return ls, nil
	} else {
		Logger.Warn("lua script not found, aborting initializing lua vm")
		return nil, nil
	}
}

// FileExists 判断文件存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CallLuaConverter 调用 Lua 脚本转换器
func CallLuaConverter(L *lua.LState, method string, raw interface{}) ([]contracts.DeviceData, error) {
	data, ok := raw.(string)
	if !ok {
		return nil, common.ProtocolDataFormatErr
	}

	// 调用脚本函数
	err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal(method),
		NRet:    1,
		Protect: true,
		Handler: nil,
	}, lua.LString(data))
	defer L.Remove(1)
	if err != nil {
		return nil, fmt.Errorf("call lua script %s function error: %s", method, err)
	}

	// 获取解析结果
	result := L.Get(-1).String()
	res := make([]contracts.DeviceData, 0)
	err = json.Unmarshal([]byte(result), &res)
	return res, err
}

// CallLuaEncodeConverter 调用 Lua 脚本编码转换器
func CallLuaEncodeConverter(L *lua.LState, deviceName string, raw interface{}) (string, error) {
	data, ok := raw.(string)
	if !ok {
		return "", common.ProtocolDataFormatErr
	}

	// 调用脚本函数
	err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("encode"),
		NRet:    1,
		Protect: true,
		Handler: nil,
	}, lua.LString(deviceName), lua.LString(data))
	defer L.Remove(1)
	if err != nil {
		return "", fmt.Errorf("call lua script encode function error: %s", err)
	}

	// 获取解析结果
	result := L.Get(-1).String()
	return result, err
}
