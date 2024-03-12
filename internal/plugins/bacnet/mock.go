package bacnet

import (
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/btypes"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

func mockRead(L *lua.LState, data btypes.MultiplePropertyData) (btypes.MultiplePropertyData, error) {
	var objects []btypes.Object
	for _, object := range data.Objects {
		data, _ := helper.CallLuaMethod(L, "mockRead", lua.LString(object.DeviceSn), lua.LString(object.Name))
		objects = append(objects, btypes.Object{
			Properties: []btypes.Property{
				{
					Type: btypes.PROP_PRESENT_VALUE,
					Data: data,
				},
			},
			DeviceSn: object.DeviceSn,
			Name:     object.Name,
		})
	}
	out := btypes.MultiplePropertyData{
		Objects:    objects,
		ErrorCode:  data.ErrorCode,
		ErrorClass: data.ErrorClass,
	}
	return out, nil
}

func mockWrite(L *lua.LState, deviceSn, pointName string, value interface{}) error {
	result, err := helper.CallLuaMethod(L, "mockWrite", lua.LString(deviceSn), lua.LString(pointName), lua.LString(fmt.Sprint(value)))
	if err == nil {
		helper.Logger.Info("mockWrite result", zap.Any("result", result))
	}
	return err
}
