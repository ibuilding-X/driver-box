package library

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper/utils"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/lua"
	glua "github.com/yuin/gopher-lua"
	"path"
	"sync"
)

// 同步锁
var deviceDriverLock sync.Mutex

type DeviceDriver struct {
	drivers *sync.Map
}

// 加载指定key的驱动
func (device *DeviceDriver) loadLibrary(driverKey string) (*glua.LState, error) {
	deviceDriverLock.Lock()
	defer deviceDriverLock.Unlock()
	cache, ok := device.drivers.Load(driverKey)
	if ok {
		return cache.(*glua.LState), nil
	}
	L, err := lua.InitLuaVM(path.Join(config.ResourcePath, baseDir, string(deviceDriver), driverKey+".lua"))
	if err != nil {
		return nil, err
	}
	device.drivers.Store(driverKey, L)
	return L, nil
}

// 设备下行指令编码，该接口试下如下功能：
// 1. 写操作时，对点位值进行加工
// 2. 针对点位A发起的读写操作，通过编码可变更为点位B
// 3. 对单点位发起的读写请求，通过编码可扩展为多点位。例如：执行空开的开关操作，会先触发解锁，再执行开关行为。
func (device *DeviceDriver) DeviceEncode(driverKey string, req DeviceEncodeRequest) *DeviceEncodeResult {
	cache, ok := device.drivers.Load(driverKey)
	var L *glua.LState
	if !ok {
		l, err := device.loadLibrary(driverKey)
		if err != nil {
			return &DeviceEncodeResult{Error: err}
		}
		L = l
	} else {
		L = cache.(*glua.LState)
	}

	points := L.NewTable()
	for _, point := range req.Points {
		pointData := L.NewTable()
		pointData.RawSetString("name", glua.LString(point.PointName))
		if req.Mode == plugin.WriteMode {
			//经过 ConvPointType 加工，数据类型一定属于string、float64、int64之一
			switch v := point.Value.(type) {
			case string:
				pointData.RawSetString("value", glua.LString(v))
			case float64:
				pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(v)))
			case int64:
				pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(v)))
			default:
				return &DeviceEncodeResult{Error: fmt.Errorf("unsupported point value type: %T", v)}
			}
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
func (device *DeviceDriver) DeviceDecode(driverKey string, req DeviceDecodeRequest) *DeviceDecodeResult {
	cache, ok := device.drivers.Load(driverKey)
	var L *glua.LState
	if !ok {
		l, err := device.loadLibrary(driverKey)
		if err != nil {
			return &DeviceDecodeResult{Error: err}
		}
		L = l
	} else {
		L = cache.(*glua.LState)
	}
	points := L.NewTable()
	for _, point := range req.Points {
		pointData := L.NewTable()
		pointData.RawSetString("name", glua.LString(point.PointName))
		switch v := point.Value.(type) {
		case string:
			pointData.RawSetString("value", glua.LString(v))
		case int8, int16, int32, int64, int, uint, uint8, uint16, uint32, uint64:
			intValue, e := utils.Conv2Int64(v)
			if e != nil {
				return &DeviceDecodeResult{Error: e}
			}
			pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(intValue)))
		case float32, float64:
			floatValue, e := utils.Conv2Float64(v)
			if e != nil {
				return &DeviceDecodeResult{Error: e}
			}
			pointData.RawSetString("value", glua.LVAsNumber(glua.LNumber(floatValue)))
		default:
			return &DeviceDecodeResult{Error: fmt.Errorf("unsupported point value type: %T", v)}
		}
		points.Append(pointData)
	}
	result, e := lua.CallLuaMethodV2(L, "decode", glua.LString(req.DeviceId), points)
	if e != nil {
		return &DeviceDecodeResult{Error: e}
	}
	res := make([]plugin.PointData, 0)
	result.ForEach(func(key, value glua.LValue) {
		res = append(res, plugin.PointData{
			PointName: glua.LVAsString(key),
			Value:     glua.LVAsString(value),
		})
	})
	return &DeviceDecodeResult{
		Points: res,
		Error:  e,
	}
}

// 卸载驱动
func (device *DeviceDriver) UnloadDeviceDrivers() {
	temp := device.drivers
	device.drivers = &sync.Map{}
	temp.Range(func(key, value interface{}) bool {
		lua.Close(value.(*glua.LState))
		return true
	})
}

// 设备驱动编码请求
type DeviceEncodeRequest struct {
	DeviceId string // 设备ID
	Mode     plugin.EncodeMode
	Points   []plugin.PointData
}

// 设备驱动编码结果
type DeviceEncodeResult struct {
	Points []plugin.PointData `json:"points"`
	Error  error
}

// 设备驱动解码请求
type DeviceDecodeRequest struct {
	DeviceId string             `json:"id"` // 设备ID
	Points   []plugin.PointData `json:"points"`
}

// 设备驱动解码结果
type DeviceDecodeResult struct {
	//解码结果
	Points []plugin.PointData `json:"points"`
	//解码错误信息
	Error error `json:"error"`
}
