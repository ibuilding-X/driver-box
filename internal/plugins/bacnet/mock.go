package bacnet

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/btypes"
	lua "github.com/yuin/gopher-lua"
)

func mockRead(L *lua.LState, data btypes.MultiplePropertyData) (out btypes.MultiplePropertyData, err error) {
	objects := make([]btypes.Object, len(data.Objects))
	for _, object := range data.Objects {
		m := make(map[string]string)
		m["deviceSn"] = object.DeviceSn
		m["pointName"] = object.Name
		bytes, _ := json.Marshal(m)
		data, _ := helper.CallLuaMethod(L, "mockRead", string(bytes))
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
	out = btypes.MultiplePropertyData{
		Objects:    objects,
		ErrorCode:  data.ErrorCode,
		ErrorClass: data.ErrorClass,
	}
	return
}

func mockWrite(L *lua.LState, deviceSn, pointName string, value interface{}) error {
	m := make(map[string]interface{})
	m["deviceSn"] = deviceSn
	m["pointName"] = pointName
	m["value"] = value
	bytes, _ := json.Marshal(m)
	_, err := helper.CallLuaMethod(L, "mockWrite", string(bytes))
	return err
}
