package library

import (
	lua "github.com/yuin/gopher-lua"
)

var Drivers = make(map[string]*lua.LState)

// 设备下行指令编码，该接口试下如下功能：
// 1. 写操作时，对点位值进行加工
// 2. 针对点位A发起的读写操作，通过编码可变更为点位B
// 3. 对单点位发起的读写请求，通过编码可扩展为多点位。例如：执行空开的开关操作，会先触发解锁，再执行开关行为。
func DeviceEncode(driverKey string, req DeviceEncodeRequest) *DeviceEncodeResult {
	//L := Drivers[driverKey]
	//points := L.NewTable()
	//for _, point := range req.Points {
	//	pointData := L.NewTable()
	//	pointData.RawSetString("name", lua.LString(point.PointName))
	//	if req.Mode == plugin.WriteMode {
	//		b, e := json.Marshal(point.Value)
	//		if e != nil {
	//			return &DeviceEncodeResult{Error: e}
	//		}
	//		pointData.RawSetString("value", lua.LString(b))
	//	}
	//	points.Append(pointData)
	//}
	//result, e := helper.CallLuaMethod(L, "encode", lua.LString(req.DeviceId), lua.LString(req.Mode), points)
	//if e != nil {
	//	return &DeviceEncodeResult{Error: e}
	//}
	//res := make([]plugin.PointData, 0)
	//e = json.Unmarshal([]byte(result), &res)
	//return &DeviceEncodeResult{
	//	Points: res,
	//	Error:  e,
	//}
	return nil
}

// 设备上行数据解码，该接口主要功能如下：
// 1. 对读到的数据进行点位值加工
// 2. 将读到的点位值，同步到本设备的另外一个点位上
func DeviceDecode(driverKey string, req DeviceDecodeRequest) *DeviceDecodeResult {
	//L := Drivers[driverKey]
	//points := L.NewTable()
	//for _, point := range req.Points {
	//	pointData := L.NewTable()
	//	pointData.RawSetString("name", lua.LString(point.PointName))
	//	b, e := json.Marshal(point.Value)
	//	if e != nil {
	//		return &DeviceDecodeResult{Error: e}
	//	}
	//	pointData.RawSetString("value", lua.LString(b))
	//	points.Append(pointData)
	//}
	//result, e := helper.CallLuaMethod(L, "decode", lua.LString(req.DeviceId), points)
	//if e != nil {
	//	return &DeviceDecodeResult{Error: e}
	//}
	//res := make([]plugin.PointData, 0)
	//e = json.Unmarshal([]byte(result), &res)
	//return &DeviceDecodeResult{
	//	Points: res,
	//	Error:  e,
	//}
	return nil
}
