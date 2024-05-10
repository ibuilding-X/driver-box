package library

import (
	"encoding/json"
	"errors"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/lua"
	glua "github.com/yuin/gopher-lua"
	"path"
)

var Drivers = make(map[string]*glua.LState)

type Type string

const (
	//设备层驱动
	DeviceDriver Type = "driver"
	//物模型
	DeviceModel Type = "model"
	//协议层驱动
	ProtocolDriver Type = "protocol"
)

// 加载指定key的驱动
func LoadLibrary(libType Type, driverKey string) error {
	switch libType {
	case DeviceDriver:
		L, err := lua.InitLuaVM(path.Join(config.ResourcePath, "library", string(libType), driverKey+".lua"))
		if err != nil {
			return err
		}
		Drivers[driverKey] = L
	default:
		return errors.New("not support library type")
	}
	return nil
}

// 设备下行指令编码，该接口试下如下功能：
// 1. 写操作时，对点位值进行加工
// 2. 针对点位A发起的读写操作，通过编码可变更为点位B
// 3. 对单点位发起的读写请求，通过编码可扩展为多点位。例如：执行空开的开关操作，会先触发解锁，再执行开关行为。
func DeviceEncode(driverKey string, req DeviceEncodeRequest) *DeviceEncodeResult {
	L := Drivers[driverKey]
	points := L.NewTable()
	for _, point := range req.Points {
		pointData := L.NewTable()
		pointData.RawSetString("name", glua.LString(point.PointName))
		if req.Mode == plugin.WriteMode {
			b, e := json.Marshal(point.Value)
			if e != nil {
				return &DeviceEncodeResult{Error: e}
			}
			pointData.RawSetString("value", glua.LString(b))
		}
		points.Append(pointData)
	}
	result, e := lua.CallLuaMethod(L, "encode", glua.LString(req.DeviceId), glua.LString(req.Mode), points)
	if e != nil {
		return &DeviceEncodeResult{Error: e}
	}
	res := make([]plugin.PointData, 0)
	e = json.Unmarshal([]byte(result), &res)
	return &DeviceEncodeResult{
		Points: res,
		Error:  e,
	}
}

// 设备上行数据解码，该接口主要功能如下：
// 1. 对读到的数据进行点位值加工
// 2. 将读到的点位值，同步到本设备的另外一个点位上
func DeviceDecode(driverKey string, req DeviceDecodeRequest) *DeviceDecodeResult {
	L := Drivers[driverKey]
	points := L.NewTable()
	for _, point := range req.Points {
		pointData := L.NewTable()
		pointData.RawSetString("name", glua.LString(point.PointName))
		b, e := json.Marshal(point.Value)
		if e != nil {
			return &DeviceDecodeResult{Error: e}
		}
		pointData.RawSetString("value", glua.LString(b))
		points.Append(pointData)
	}
	result, e := lua.CallLuaMethod(L, "decode", glua.LString(req.DeviceId), points)
	if e != nil {
		return &DeviceDecodeResult{Error: e}
	}
	res := make([]plugin.PointData, 0)
	e = json.Unmarshal([]byte(result), &res)
	return &DeviceDecodeResult{
		Points: res,
		Error:  e,
	}
}

// 卸载驱动
func UnloadDeviceDrivers() {
	temp := Drivers
	Drivers = make(map[string]*glua.LState)
	for _, L := range temp {
		lua.Close(L)
	}
}
