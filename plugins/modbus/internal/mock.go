package internal

import (
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/fileutil"
	"github.com/ibuilding-x/driver-box/pkg/luautil"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

var ls *lua.LState

func InitMockLua(key string) {
	if ls == nil {
		path := filepath.Join(helper.EnvConfig.ConfigPath, key, "converter.lua")
		if fileutil.FileExists(path) {
			l, err := luautil.InitLuaVM(path)
			if err != nil {
				helper.Logger.Error("init lua vm error", zap.Error(err))
				return
			}
			ls = l
		}
	}
}
func (c *connector) mockRead(slaveId uint8, registerType string, address, quantity uint16) (values []uint16, err error) {
	if ls == nil {
		return nil, errors.New("lua vm is nil")
	}
	mockData, err := luautil.CallLuaMethod(ls, "mockRead", lua.LNumber(slaveId), lua.LString(registerType), lua.LNumber(address), lua.LNumber(quantity))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(mockData), &values)
	return
}

func (c *connector) mockWrite(slaveID uint8, registerType primaryTable, address uint16, values []uint16) error {
	if ls == nil {
		return errors.New("lua vm is nil")
	}
	valueTable := ls.NewTable()
	for _, v := range values {
		valueTable.Append(lua.LNumber(v))
	}
	result, err := luautil.CallLuaMethod(ls, "mockWrite", lua.LNumber(slaveID), lua.LString(registerType), lua.LNumber(address), valueTable)
	if err == nil {
		helper.Logger.Info("mockWrite result", zap.Any("result", result))
	}
	return err
}
