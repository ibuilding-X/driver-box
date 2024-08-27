package lua

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cjoudrey/gluahttp"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/logger"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	luajson "layeh.com/gopher-json"
	"net/http"
	"sync"
)

// 缓存Lua虚拟机的锁
var luaLocks = sync.Map{}

// InitLuaVM 编译 lua 脚本
func InitLuaVM(filePath string) (*lua.LState, error) {
	if !common.FileExists(filePath) {
		logger.Logger.Warn("lua script not found, aborting initializing lua vm", zap.Any("filePath", filePath))
		return nil, errors.New("lua script not found")
	}
	ls := lua.NewState(lua.Options{
		RegistryMaxSize: 128,
	})
	// 预加载模块（json、http、storage）
	ls.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	luajson.Preload(ls)
	ls.PreloadModule("driverbox", LuaModuleInstance.Loader)
	// 文件路径
	// 脚本解析
	err := ls.DoFile(filePath)
	if err != nil {
		return nil, err
	}
	//注册同步锁
	luaLocks.Store(ls, &sync.Mutex{})
	return ls, nil
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
			logger.Logger.Error("call lua script error", zap.Any("method", method), zap.Any("args", args), zap.Any("error", err))
		}
	}()

	lock, ok := luaLocks.Load(L)
	if !ok {
		return "", errors.New("lua VM not exists")
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
		return "", err
	}
	return L.Get(-1).String(), nil
}

// CallLuaEncodeConverter 调用 Lua 脚本编码转换器
func CallLuaEncodeConverter(L *lua.LState, deviceSn string, raw interface{}) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			logger.Logger.Error("call lua script error", zap.Any("error", err))
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
