package helper

import (
	"encoding/json"
	"fmt"
	"github.com/cjoudrey/gluahttp"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var runLock = &sync.Mutex{}

// InitLuaVM 编译 lua 脚本
func InitLuaVM(scriptDir string) (*lua.LState, error) {
	ls := lua.NewState(lua.Options{
		RegistryMaxSize: 128,
	})
	// 预加载模块（json、http、storage）
	ls.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	luajson.Preload(ls)
	ls.PreloadModule("driverbox", LuaModuleInstance.Loader)
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
func CallLuaConverter(L *lua.LState, method string, raw interface{}) ([]plugin.DeviceData, error) {
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
	res := make([]plugin.DeviceData, 0)
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

// SafeCallLuaFunc 安全调用 lua 函数，通过锁机制独占时间片
func SafeCallLuaFunc(L *lua.LState, method string) error {
	runLock.Lock()
	defer runLock.Unlock()

	L.Push(L.GetGlobal(method))
	if err := L.PCall(0, 0, nil); err != nil {
		return fmt.Errorf("call lua script %s function error: %s", method, err)
	}

	return nil
}
