package library

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/lua"
	glua "github.com/yuin/gopher-lua"
	"path"
)

type ProtocolDriver struct {
	drivers map[string]*glua.LState
}

// 加载指定key的驱动
func (device *ProtocolDriver) LoadLibrary(driverKey string) error {
	L, err := lua.InitLuaVM(path.Join(config.ResourcePath, baseDir, string(protocolDriver), driverKey+".lua"))
	if err != nil {
		return err
	}
	device.drivers[driverKey] = L
	return nil
}

// 执行制定的方法
func (device *ProtocolDriver) Execute(driverKey string, luaMethod string, param string) (string, error) {
	L := device.drivers[driverKey]
	return lua.CallLuaMethod(L, luaMethod, glua.LString(param))
}

// 设备下行指令编码，该接口试下如下功能：
// 1. 写操作时，对点位值进行加工
// 2. 针对点位A发起的读写操作，通过编码可变更为点位B
// 3. 对单点位发起的读写请求，通过编码可扩展为多点位。例如：执行空开的开关操作，会先触发解锁，再执行开关行为。
func (device *ProtocolDriver) Encode(driverKey string, req ProtocolEncodeRequest) (string, error) {
	L := device.drivers[driverKey]
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

// 设备上行数据解码，该接口主要功能如下：
// 1. 对读到的数据进行点位值加工
// 2. 将读到的点位值，同步到本设备的另外一个点位上
func (device *ProtocolDriver) Decode(driverKey string, req any) ([]plugin.DeviceData, error) {
	L := device.drivers[driverKey]
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
