package library

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/event"
	"github.com/ibuilding-x/driver-box/driverbox/internal/lua"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	glua "github.com/yuin/gopher-lua"
)

type ProtocolDriver struct {
	drivers map[string]*glua.LState
}

// 加载指定key的驱动
func (device *ProtocolDriver) LoadLibrary(protocolKey string) error {
	L, err := lua.InitLuaVM(path.Join(config.ResourcePath, baseDir, string(protocolDriver), protocolKey+".lua"))
	if err != nil {
		return err
	}
	device.drivers[protocolKey] = L
	return nil
}

// 执行制定的方法
func (device *ProtocolDriver) Execute(protocolKey string, luaMethod string, param string) (string, error) {
	L := device.drivers[protocolKey]
	return lua.CallLuaMethod(L, luaMethod, glua.LString(param))
}

// 设备下行指令编码，该接口试下如下功能：
// 1. 写操作时，对点位值进行加工
// 2. 针对点位A发起的读写操作，通过编码可变更为点位B
// 3. 对单点位发起的读写请求，通过编码可扩展为多点位。例如：执行空开的开关操作，会先触发解锁，再执行开关行为。
func (device *ProtocolDriver) Encode(protocolKey string, req ProtocolEncodeRequest) (string, error) {
	L := device.drivers[protocolKey]
	points := L.NewTable()
	for _, point := range req.Points {
		pointData := L.NewTable()
		pointData.RawSetString("name", glua.LString(point.PointName))
		if req.Mode == plugin.WriteMode {
			//经过ConvPointType加工，数据类型一定属于string、float64、int64之一
			switch v := point.Value.(type) {
			case string:
				pointData.RawSetString("value", glua.LString(v))
			case float64:
				pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(v)))
			case int64:
				pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(v)))
			default:
				return "", fmt.Errorf("unsupported point value type: %T", v)
			}
		}
		points.Append(pointData)
	}
	return lua.CallLuaMethod(L, "encode", glua.LString(req.DeviceId), glua.LString(req.Mode), points)
}

func (device *ProtocolDriver) EncodeV2(protocolKey string, req ProtocolEncodeRequest) (*glua.LTable, error) {
	L := device.drivers[protocolKey]
	points := L.NewTable()
	for _, point := range req.Points {
		pointData := L.NewTable()
		pointData.RawSetString("name", glua.LString(point.PointName))
		if req.Mode == plugin.WriteMode {
			//经过ConvPointType加工，数据类型一定属于string、float64、int64之一
			switch v := point.Value.(type) {
			case string:
				pointData.RawSetString("value", glua.LString(v))
			case float64:
				pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(v)))
			case int64:
				pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(v)))
			default:
				return nil, fmt.Errorf("unsupported point value type: %T", v)
			}
		}
		points.Append(pointData)
	}
	return lua.CallLuaMethodV2(L, "encode", glua.LString(req.DeviceId), glua.LString(req.Mode), points)
}

// Deprecated:
// 设备上行数据解码，该接口主要功能如下：
// 1. 对读到的数据进行点位值加工
// 2. 将读到的点位值，同步到本设备的另外一个点位上
func (device *ProtocolDriver) Decode(protocolKey string, req any) ([]plugin.DeviceData, error) {
	L := device.drivers[protocolKey]
	bytes, e := json.Marshal(req)
	if e != nil {
		return nil, e
	}
	result, e := lua.CallLuaMethod(L, "decode", glua.LString(bytes))
	if e != nil {
		return nil, e
	}
	res := make([]plugin.DeviceData, 0)
	e = json.Unmarshal([]byte(result), &res)
	return res, e
}

func (device *ProtocolDriver) DecodeV2(protocolKey string, param func(L *glua.LState) *glua.LTable) ([]plugin.DeviceData, error) {
	L := device.drivers[protocolKey]
	result, e := lua.CallLuaMethodV2(L, "decode", param(L))
	if e != nil {
		return nil, e
	}
	if result == nil || result.Type() != glua.LTTable {
		return nil, fmt.Errorf("decode result is not table")
	}
	res := make([]plugin.DeviceData, 0)
	result.ForEach(func(key, value glua.LValue) {
		table := value.(*glua.LTable)
		deviceData := plugin.DeviceData{
			ID:     glua.LVAsString(table.RawGetString("id")),
			Values: []plugin.PointData{},
			Events: []event.Data{},
		}
		values := table.RawGet(glua.LString("values"))
		if values.Type() == glua.LTTable {
			values.(*glua.LTable).ForEach(func(key, value glua.LValue) {
				point := value.(*glua.LTable)
				pointData := plugin.PointData{
					PointName: glua.LVAsString(point.RawGetString("name")),
					Value:     glua.LVAsString(point.RawGetString("value")),
				}
				deviceData.Values = append(deviceData.Values, pointData)
			})
		}
		events := table.RawGet(glua.LString("events"))
		if events.Type() == glua.LTTable {
			//事件解析
			events.(*glua.LTable).ForEach(func(key, value glua.LValue) {
				devEvent := value.(*glua.LTable)
				deviceData.Events = append(deviceData.Events, event.Data{
					Code:  glua.LVAsString(devEvent.RawGetString("event")),
					Value: convertLuaValue(devEvent.RawGetString("value")),
				})
			})
		}
		res = append(res, deviceData)
	})
	return res, e
}

// 卸载驱动
func (device *ProtocolDriver) UnloadDeviceDrivers() {
	temp := device.drivers
	device.drivers = make(map[string]*glua.LState)
	for _, L := range temp {
		lua.Close(L)
	}
}

// 设备驱动编码请求
type ProtocolEncodeRequest struct {
	DeviceId string // 设备ID
	Mode     plugin.EncodeMode
	Points   []plugin.PointData
}
