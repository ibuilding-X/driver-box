package knx

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/plugins/knx/knx/cemi"
	"github.com/spf13/cast"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
	"strings"
)

type primaryTable string

const (
	Coil            primaryTable = "COIL"             // 线圈
	DiscreteInput   primaryTable = "DISCRETE_INPUT"   // 离散输入
	InputRegister   primaryTable = "INPUT_REGISTER"   // 离散寄存器
	HoldingRegister primaryTable = "HOLDING_REGISTER" // 保持寄存器
)

// Adapter 协议适配器
type adapter struct {
	scriptDir string // 脚本目录名称
	ls        *lua.LState
}

// transportationData 通讯数据（编码返回、解码输入数据格式）
type transportationData struct {
	DeviceName string            // 设备名称
	Mode       plugin.EncodeMode // 读取模式
	Address    string            `json:"address"`
	Dpt        string            `json:"dpt"`
	PointName  string            // 点位名称
	Value      interface{}       // 写入值，仅当 write 模式时使用
}

// Value 设定值及掩码
type Value struct {
	Value uint16 `json:"value"` // 寄存器值
	Mask  uint16 `json:"mask"`  // 掩码  为 0 时无效
}

type pointTargetValue struct {
	Name        string      `json:"name"`        // 点位名称
	TargetValue interface{} `json:"targetValue"` // 需要设定的值
	Values      []Value     `json:"values"`      // 根据rawType编码出的寄存器值
}

type extendProps struct {
	RegisterType string         `json:"primaryTable"`
	StartAddress uint16         `json:"startAddress"`
	Quantity     uint16         `json:"quantity"`
	RawType      string         `json:"rawType"`
	Virtual      bool           `json:"virtual"`
	Points       []string       `json:"points"`
	ByteSwap     bool           `json:"byteSwap"`
	WordSwap     bool           `json:"wordSwap"`
	addr         cemi.GroupAddr `json:"-"`
}

// Encode 编码数据
func (a *adapter) Encode(deviceName string, mode plugin.EncodeMode, value plugin.PointData) (res interface{}, err error) {
	// 获取设备数据
	device, ok := helper.CoreCache.GetDeviceByDeviceAndPoint(deviceName, value.PointName)
	if !ok {
		return nil, fmt.Errorf("not found device, deviceName is %s", deviceName)
	}
	// 响应结果
	result := transportationData{
		DeviceName: deviceName,
		Mode:       mode,
		Address:    device.Protocol[ATTRI_ADDRESS],
		Dpt:        device.Protocol[ATTRI_DPT],
		PointName:  value.PointName,
	}
	point, ok := helper.CoreCache.GetPointByDevice(deviceName, value.PointName)
	if !ok {
		return nil, fmt.Errorf("not found point from core config, deviceName is %s, point name is %s", deviceName, value.PointName)
	}
	if mode == plugin.WriteMode {
		ptv := &pointTargetValue{
			Name:        value.PointName,
			TargetValue: value.Value,
		}
		ext, err := getExtendProps(deviceName, point.Name)
		if err != nil {
			return nil, fmt.Errorf("extend prop parsed error: %s", err.Error())
		}
		// 根据配置数据类型进行编码
		ptv.encodeRawValue(ext.RegisterType, ext.RawType, ext.ByteSwap, ext.WordSwap)

		// lua脚本存在时调用lua脚本进行编码
		var encodedData string
		if a.scriptExists() {
			bytes, _ := json.Marshal(*ptv)
			encodedData, err = helper.CallLuaEncodeConverter(a.ls, deviceName, string(bytes))
			fmt.Printf("----encode----%+v\n", encodedData)
			if err != nil {
				return nil, err
			}
			var eptv pointTargetValue
			err = json.Unmarshal([]byte(encodedData), &eptv)
			if err != nil {
				return nil, fmt.Errorf("encode error for device %s, original table is %+v",
					deviceName, eptv)
			}
			if eptv.Name == ptv.Name {
				//result.Values = eptv.Values
			} else {
				//result.Values = ptv.Values
			}
		} else {
			//result.Values = ptv.Values
		}
	}
	// 返回协议数据
	return result, nil
}

// encodeRawValue 根据rawType将原始值转化为uint16数组 按照数据顺序排列
func (ptv *pointTargetValue) encodeRawValue(registerType, rawType string, byteSwap, wordSwap bool) {
	switch strings.ToUpper(registerType) {
	case string(Coil):
		ptv.Values = []Value{{
			Value: cast.ToUint16(ptv.TargetValue),
		}}
	default:
		switch strings.ToUpper(rawType) {
		case strings.ToUpper(common.ValueTypeUint16):
			ptv.Values = []Value{{
				Value: BytesToUint16(BIG_ENDIAN, Uint16ToBytes(Endianness(byteSwap), cast.ToUint16(ptv.TargetValue))),
			}}
		case strings.ToUpper(common.ValueTypeInt16):
			conv2Int64, _ := helper.Conv2Int64(ptv.TargetValue)
			val := conv2Int64 & 0xFFFF
			ptv.Values = []Value{{
				Value: BytesToUint16(BIG_ENDIAN, Uint16ToBytes(Endianness(byteSwap), cast.ToUint16(uint16(val)))),
			}}
		case strings.ToUpper(common.ValueTypeUint32):
			uint16s := BytesToUint16s(BIG_ENDIAN, Uint32ToBytes(Endianness(byteSwap), WordOrder(wordSwap), cast.ToUint32(ptv.TargetValue)))
			ptv.Values = []Value{
				{Value: uint16s[0]},
				{Value: uint16s[1]},
			}
		case strings.ToUpper(common.ValueTypeInt32):
			conv2Int64, _ := helper.Conv2Int64(ptv.TargetValue)
			val := conv2Int64 & 0xFFFFFFFF
			uint16s := BytesToUint16s(BIG_ENDIAN, Uint32ToBytes(Endianness(byteSwap), WordOrder(wordSwap), cast.ToUint32(uint32(val))))
			ptv.Values = []Value{
				{Value: uint16s[0]},
				{Value: uint16s[1]},
			}
		case strings.ToUpper(common.ValueTypeUint64):
			uint16s := BytesToUint16s(BIG_ENDIAN, Uint64ToBytes(Endianness(byteSwap), WordOrder(wordSwap), cast.ToUint64(ptv.TargetValue)))
			ptv.Values = []Value{
				{Value: uint16s[0]},
				{Value: uint16s[1]},
				{Value: uint16s[2]},
				{Value: uint16s[3]},
			}
		case strings.ToUpper(common.ValueTypeInt64):
			val, _ := helper.Conv2Int64(ptv.TargetValue)
			uint16s := BytesToUint16s(BIG_ENDIAN, Uint64ToBytes(Endianness(byteSwap), WordOrder(wordSwap), cast.ToUint64(uint64(val))))
			ptv.Values = []Value{
				{Value: uint16s[0]},
				{Value: uint16s[1]},
				{Value: uint16s[2]},
				{Value: uint16s[3]},
			}
		case strings.ToUpper(common.ValueTypeFloat32):
			uint16s := BytesToUint16s(BIG_ENDIAN, Float32ToBytes(Endianness(byteSwap), WordOrder(wordSwap), cast.ToFloat32(ptv.TargetValue)))
			ptv.Values = []Value{
				{Value: uint16s[0]},
				{Value: uint16s[1]},
			}
		case strings.ToUpper(common.ValueTypeFloat64):
			uint16s := BytesToUint16s(BIG_ENDIAN, Float64ToBytes(Endianness(byteSwap), WordOrder(wordSwap), cast.ToFloat64(ptv.TargetValue)))
			ptv.Values = []Value{
				{Value: uint16s[0]},
				{Value: uint16s[1]},
				{Value: uint16s[2]},
				{Value: uint16s[3]},
			}
		case strings.ToUpper(common.ValueTypeString):
			uint16s := BytesToUint16s(Endianness(byteSwap), []byte(cast.ToString(ptv.TargetValue)))
			var values []Value
			for _, v := range uint16s {
				values = append(values, Value{Value: v})
			}
			ptv.Values = values
		}
	}
}

// Decode 解码数据
func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	//deviceRawData := raw.(knx.GroupEvent)
	//deviceRawDataBytes, _ := json.Marshal(deviceRawData)
	//lo("raw data: %s",deviceRawDataBytes)
	//if a.scriptExists() {
	//	return helper.CallLuaConverter(a.ls, "decode", string(deviceRawDataBytes))
	//} else {
	//	// 当前设备未被lua解析
	//	var pointDatalist []plugin.PointData
	//	for _, prv := range deviceRawData.PointRawValues {
	//		pointDatalist = append(pointDatalist, plugin.PointData{
	//			PointName: prv.Name,
	//			Value:     prv.Value,
	//		})
	//	}
	//	res = append(res, plugin.DeviceData{
	//		DeviceName: deviceRawData.DeviceName,
	//		Values:     pointDatalist,
	//	})
	//}
	return
}

// scriptExists 判断lua脚本是否存在
func (a *adapter) scriptExists() bool {
	scriptPath := filepath.Join(common.CoreConfigPath, a.scriptDir, common.LuaScriptName)
	_, err := os.Stat(scriptPath)
	return err == nil
}
