package bacnet

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/btypes"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

func mockRead(plugin *connector, L *lua.LState, data btypes.MultiplePropertyData) error {
	for _, object := range data.Objects {
		for deviceId, pointName := range object.Points {
			mockData, e := helper.CallLuaMethod(L, "mockRead", lua.LString(deviceId), lua.LString(pointName))
			if e != nil {
				helper.Logger.Error("mockRead error", zap.Error(e))
			}
			v, e := helper.Conv2Float64(mockData)
			if e != nil {
				helper.Logger.Error("mockRead error", zap.Error(e))
				continue
			}
			resp := map[string]interface{}{
				"deviceId":  deviceId,
				"pointName": pointName,
				"value":     v,
			}
			respJson, err := json.Marshal(resp)
			_, err = callback.OnReceiveHandler(plugin, string(respJson))
			if err != nil {
				helper.Logger.Error("error bacnet callback", zap.Any("data", respJson), zap.Error(err))
			}
		}
	}
	return nil
}

func mockWrite(L *lua.LState, deviceId, pointName string, value interface{}) error {
	result, err := helper.CallLuaMethod(L, "mockWrite", lua.LString(deviceId), lua.LString(pointName), lua.LString(fmt.Sprint(value)))
	if err == nil {
		helper.Logger.Info("mockWrite result", zap.Any("result", result))
	}
	return err
}
