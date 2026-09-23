package internal

import (
	"encoding/json"
	"fmt"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/convutil"
	"github.com/ibuilding-x/driver-box/v2/pkg/luautil"
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/btypes"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

func mockRead(conn *connector, L *lua.LState, data btypes.MultiplePropertyData) error {
	var batches []plugin.DeviceData
	for _, object := range data.Objects {
		for deviceId, pointName := range object.Points {
			mockData, e := luautil.CallLuaMethod(L, "mockRead", lua.LString(deviceId), lua.LString(pointName))
			if e != nil {
				driverbox.Log().Error("mockRead error", zap.Error(e))
			}
			v, e := convutil.Float64(mockData)
			if e != nil {
				driverbox.Log().Error("mockRead error", zap.Error(e))
				continue
			}
			resp := map[string]interface{}{
				"deviceId":  deviceId,
				"pointName": pointName,
				"value":     v,
			}
			respJson, err := json.Marshal(resp)
			if err != nil {
				driverbox.Log().Error("error bacnet response encoding", zap.Error(err))
				continue
			}
			res, err := conn.Decode(string(respJson))
			if err != nil {
				driverbox.Log().Error("error bacnet callback", zap.Any("data", respJson), zap.Error(err))
			} else {
				batches = append(batches, res...)
			}
		}
	}
	driverbox.Export(batches)
	return nil
}

func mockWrite(L *lua.LState, deviceId, pointName string, value interface{}) error {
	result, err := luautil.CallLuaMethod(L, "mockWrite", lua.LString(deviceId), lua.LString(pointName), lua.LString(fmt.Sprint(value)))
	if err == nil {
		driverbox.Log().Info("mockWrite result", zap.Any("result", result))
	}
	return err
}
