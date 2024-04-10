package helper

import (
	"encoding/json"
	"fmt"
	"github.com/cjoudrey/gluahttp"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	luajson "layeh.com/gopher-json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var runLock = &sync.Mutex{}

// 缓存Lua虚拟机的锁
var luaLocks = sync.Map{}

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
	filePath := filepath.Join(EnvConfig.ConfigPath, scriptDir, common.LuaScriptName)
	if FileExists(filePath) {
		// 脚本解析
		err := ls.DoFile(filePath)
		if err != nil {
			return nil, err
		}
		//注册同步锁
		luaLocks.Store(ls, &sync.Mutex{})
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
	// 获取解析结果
	result, err := CallLuaMethod(L, method, lua.LString(data))
	if err != nil {
		return nil, err
	}
	res := make([]plugin.DeviceData, 0)
	err = json.Unmarshal([]byte(result), &res)
	return res, err
}

// 执行指定lua方法
func CallLuaMethod(L *lua.LState, method string, args ...lua.LValue) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Logger.Error("call lua script error", zap.Any("error", err))
		}
	}()

	lock, ok := luaLocks.Load(L)
	if !ok {
		return "", common.ProtocolDataFormatErr
	}
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()
	// 调用脚本函数
	err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal(method),
		NRet:    1,
		Protect: true,
		Handler: nil,
	}, args...)
	defer L.Remove(1)
	if err != nil {
		return "", fmt.Errorf("call lua script %s function error: %s", method, err)
	}
	return L.Get(-1).String(), nil
}

// CallLuaEncodeConverter 调用 Lua 脚本编码转换器
func CallLuaEncodeConverter(L *lua.LState, deviceSn string, raw interface{}) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Logger.Error("call lua script error", zap.Any("error", err))
		}
	}()

	data, ok := raw.(string)
	if !ok {
		return "", common.ProtocolDataFormatErr
	}

	lock, ok := luaLocks.Load(L)
	if !ok {
		return "", common.ProtocolDataFormatErr
	}
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()

	// 调用脚本函数
	err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("encode"),
		NRet:    1,
		Protect: true,
		Handler: nil,
	}, lua.LString(deviceSn), lua.LString(data))
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
	defer func() {
		if err := recover(); err != nil {
			Logger.Error("call lua script error", zap.Any("error", err))
		}
	}()

	runLock.Lock()
	defer runLock.Unlock()

	L.Push(L.GetGlobal(method))
	if err := L.PCall(0, 0, nil); err != nil {
		return fmt.Errorf("call lua script %s function error: %s", method, err)
	}

	return nil
}

// 关闭Lua虚拟机
func Close(L *lua.LState) {
	if L == nil {
		return
	}
	if !L.IsClosed() {
		L.Close()
	}

	luaLocks.Delete(L)
}

// scriptExists 判断lua脚本是否存在
func ScriptExists(dir string) bool {
	scriptPath := filepath.Join(EnvConfig.ConfigPath, dir, common.LuaScriptName)
	_, err := os.Stat(scriptPath)
	return err == nil
}
