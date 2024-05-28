package modbus

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

func (c *connector) mockRead(slaveId uint8, registerType string, address, quantity uint16) (values []uint16, err error) {
	mockData, e := helper.CallLuaMethod(c.plugin.ls, "mockRead", lua.LNumber(slaveId), lua.LString(registerType), lua.LNumber(address), lua.LNumber(quantity))
	e = json.Unmarshal([]byte(mockData), &values)
	return values, e
}

func (c *connector) mockWrite(slaveID uint8, registerType primaryTable, address uint16, values []uint16) error {
	valueTable := c.plugin.ls.NewTable()
	for _, v := range values {
		valueTable.Append(lua.LNumber(v))
	}
	result, err := helper.CallLuaMethod(c.plugin.ls, "mockWrite", lua.LNumber(slaveID), lua.LString(registerType), lua.LNumber(address), valueTable)
	if err == nil {
		helper.Logger.Info("mockWrite result", zap.Any("result", result))
	}
	return err
}
