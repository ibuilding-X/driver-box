package bacnet

import (
	"encoding/json"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/internal/plugins/bacnet/bacnet/network"
	"github.com/spf13/cast"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

// adapter 协议适配器
type adapter struct {
	scriptDir string // 脚本目录名称
	ls        *lua.LState
}

type bacRequest struct {
	deviceId string            // 通讯设备ID
	mode     plugin.EncodeMode // 模式
	req      interface{}       // 请求 分为读请求和写请求
}

type extends struct {
	ObjType         string `json:"objectType"`
	Ins             int    `json:"instance"`
	DefaultPriority int    `json:"defaultPriority"`
	DefaultNull     bool   `json:"defaultNull"`
	//点位采集周期
	Duration string `json:"duration"`
}

// 写命令结构体
type bacWriteCmd struct {
	plugin.PointWriteValue
	Priority  int  `json:"priority"`
	NullValue bool `json:"nullValue"`
}

func (a *adapter) BatchEncode(deviceSn string, mode plugin.EncodeMode, value []plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Encode 编码
func (a *adapter) Encode(deviceSn string, mode plugin.EncodeMode, value plugin.PointData) (res interface{}, err error) {
	device, ok := helper.CoreCache.GetDevice(deviceSn)
	if !ok {
		return nil, common.DeviceNotFoundError
	}
	deviceId := device.Properties["id"]
	point, ok := helper.CoreCache.GetPointByDevice(deviceSn, value.PointName)
	if !ok {
		return nil, common.PointNotFoundError
	}
	var ext extends
	if err = helper.Map2Struct(point.Extends, &ext); err != nil {
		return nil, err
	}
	switch mode {
	case plugin.ReadMode:
		if req, err := createReadReq(deviceSn, value.PointName, ext); err == nil {
			return bacRequest{
				req:      req,
				mode:     mode,
				deviceId: deviceId,
			}, nil
		} else {
			return nil, err
		}
	case plugin.WriteMode:
		var bwc bacWriteCmd
		v, ok := value.Value.(string)
		if !ok || v == "" || json.Unmarshal([]byte(v), &bwc) != nil {
			bwc.Value = value.Value
			bwc.Priority = ext.DefaultPriority
			bwc.NullValue = ext.DefaultNull
		}
		bwc.PointName = value.PointName
		bwc.ModelName = device.ModelName
		if a.scriptExists() {
			bytes, err := json.Marshal(bwc)
			if err != nil {
				return nil, err
			}
			result, err := helper.CallLuaEncodeConverter(a.ls, deviceSn, string(bytes))
			err = json.Unmarshal([]byte(result), &bwc)
			if err != nil {
				return nil, err
			}
		}
		//是否存在前置操作
		if len(bwc.PreOp) > 0 {
			for _, op := range bwc.PreOp {
				helper.Logger.Info("Send preOp", zap.String("deviceSn", deviceSn), zap.String("pointName", op.PointName), zap.Any("value", op.Value))
				err = core.SendSinglePoint(deviceSn, plugin.WriteMode, plugin.PointData{
					PointName: op.PointName,
					Value:     op.Value,
				})
				if err != nil {
					return nil, err
				}
			}
		}
		// begin--临时bugfix
		point, ok = helper.CoreCache.GetPointByDevice(deviceSn, bwc.PointName)
		if !ok {
			return nil, common.PointNotFoundError
		}
		var ext extends
		if err = helper.Map2Struct(point.Extends, &ext); err != nil {
			return nil, err
		}
		// end--临时bugfix
		err = bwc.transformData(ext.ObjType)
		if err != nil {
			return nil, err
		}
		if req, err := createWriteReq(bwc, ext); err == nil {
			req.DeviceSn = deviceSn
			req.PointName = bwc.PointName
			return bacRequest{
				req:      req,
				mode:     mode,
				deviceId: deviceId,
			}, nil
		} else {
			return nil, err
		}
	default:
		return nil, common.NotSupportEncode
	}
}

// createReadReq 创建读命令
func createReadReq(deviceSn, pointName string, ext extends) (btypes.MultiplePropertyData, error) {
	if !validObjType(ext.ObjType) {
		return btypes.MultiplePropertyData{}, fmt.Errorf("unsupported objType: %s", ext.ObjType)
	}
	props := []btypes.Property{
		{
			Type:       btypes.PropPresentValue,
			ArrayIndex: bacnet.ArrayAll,
		},
		{
			Type:       btypes.PROP_STATUS_FLAGS,
			ArrayIndex: bacnet.ArrayAll,
		},
	}
	rpm := btypes.MultiplePropertyData{}
	rpm.Objects = []btypes.Object{
		{
			Points: map[string]string{
				deviceSn: pointName,
			},
			ID: btypes.ObjectID{
				Type:     btypes.GetType(ext.ObjType),
				Instance: btypes.ObjectInstance(ext.Ins),
			},
			Properties: props,
		},
	}
	return rpm, nil
}

// createWriteReq 创建写命令
func createWriteReq(bwc bacWriteCmd, ext extends) (req *network.Write, err error) {
	req = &network.Write{
		ObjectID:      btypes.ObjectInstance(ext.Ins),
		ObjectType:    btypes.GetType(ext.ObjType),
		Prop:          btypes.PropPresentValue,
		WritePriority: cast.ToUint8(bwc.Priority),
		WriteNull:     bwc.NullValue,
		WriteValue:    bwc.Value,
	}
	return req, nil
}

// TransformData 写数据类型转换
func (bwc *bacWriteCmd) transformData(objType string) error {
	data := bwc.Value
	switch objType {
	// 转换枚举值
	case btypes.MultiStateValueStr, btypes.MultiStateInputStr, btypes.MultiStateOutputStr:
		val, err := cast.ToUint32E(data)
		if err != nil {
			return err
		}
		bwc.Value = val
	// 转换数值
	case btypes.AnalogInputStr, btypes.AnalogOutputStr, btypes.AnalogValueStr:
		if val, err := cast.ToFloat32E(data); err != nil {
			return err
		} else {
			bwc.Value = val
		}
	// 转换布尔值
	case btypes.BinaryInputStr, btypes.BinaryOutputStr, btypes.BinaryValueStr:
		val, err := cast.ToUint32E(data)
		if err != nil {
			return err
		}
		bwc.Value = val
	default:
		return fmt.Errorf("return result fail, none supported value type: %v", objType)
	}
	return nil
}

func validObjType(objType string) bool {
	b1 := objType == btypes.AnalogInputStr || objType == btypes.AnalogValueStr || objType == btypes.AnalogOutputStr || objType == btypes.LargeAnalogValueStr
	b2 := objType == btypes.BinaryInputStr || objType == btypes.BinaryValueStr || objType == btypes.BinaryOutputStr
	b3 := objType == btypes.MultiStateInputStr || objType == btypes.MultiStateValueStr || objType == btypes.MultiStateOutputStr
	return b1 || b2 || b3
}

// Decode 解码
func (a *adapter) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	if a.scriptExists() {
		return helper.CallLuaConverter(a.ls, "decode", raw)
	} else {
		rawJson := raw.(string)
		var resp readResponse
		err := json.Unmarshal([]byte(rawJson), &resp)
		if err != nil {
			return nil, err
		}
		// 当前设备未被lua解析
		pointDatalist := []plugin.PointData{{
			PointName: resp.PointName,
			Value:     resp.Value,
		}}
		res = append(res, plugin.DeviceData{
			SN:     resp.DeviceSn,
			Values: pointDatalist,
		})
	}

	return
}

// scriptExists 判断lua脚本是否存在
func (a *adapter) scriptExists() bool {
	scriptPath := filepath.Join(helper.EnvConfig.ConfigPath, a.scriptDir, common.LuaScriptName)
	_, err := os.Stat(scriptPath)
	return err == nil
}
