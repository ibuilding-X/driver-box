package internal

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/convutil"
	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet"
	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet/btypes"
	"github.com/ibuilding-x/driver-box/plugins/bacnet/internal/bacnet/network"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

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

// PointWriteValue 点位写操作的结构体
type PointWriteValue struct {
	// PointName 点位名称
	PointName string `json:"pointName"`
	// Value 点位值
	Value interface{} `json:"value"`
	//模型名称，某些驱动解析需要根据模型作区分
	ModelName string `json:"modelName"`
	//前置操作，例如空开要先解锁，空调要先开机
	PreOp []PointWriteValue `json:"preOp"`
}

// 写命令结构体
type bacWriteCmd struct {
	PointWriteValue
	Priority  int  `json:"priority"`
	NullValue bool `json:"nullValue"`
}

// Encode 编码
func (c *connector) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	device, ok := driverbox.CoreCache().GetDevice(deviceSn)
	if !ok {
		return nil, errors.New("device not found error")
	}
	deviceId := device.Properties["id"]

	switch mode {
	case plugin.ReadMode:
		if len(values) != 1 {
			return nil, plugin.NotSupportEncode
		}
		value := values[0]
		point, ok := driverbox.CoreCache().GetPointByDevice(deviceSn, value.PointName)
		if !ok {
			return nil, errors.New("point not found error")
		}
		var ext extends
		if err = convutil.Struct(point, &ext); err != nil {
			return nil, err
		}

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
		writeReqs := make([]*network.Write, 0)
		for _, value := range values {
			point, ok := driverbox.CoreCache().GetPointByDevice(deviceSn, value.PointName)
			if !ok {
				return nil, errors.New("point not found error")
			}
			var ext extends
			if err = convutil.Struct(point, &ext); err != nil {
				return nil, err
			}
			var bwc bacWriteCmd
			v, ok := value.Value.(string)
			if !ok || v == "" || json.Unmarshal([]byte(v), &bwc) != nil {
				bwc.Value = value.Value
				bwc.Priority = ext.DefaultPriority
				bwc.NullValue = ext.DefaultNull
			}
			bwc.PointName = value.PointName
			bwc.ModelName = device.ModelName
			//if c.plugin.ls != nil {
			//	bytes, err := json.Marshal(bwc)
			//	if err != nil {
			//		return nil, err
			//	}
			//	result, err := luautil.CallLuaEncodeConverter(c.plugin.ls, deviceSn, string(bytes))
			//	err = json.Unmarshal([]byte(result), &bwc)
			//	if err != nil {
			//		return nil, err
			//	}
			//}
			//是否存在前置操作
			if len(bwc.PreOp) > 0 {
				for _, op := range bwc.PreOp {
					driverbox.Log().Info("Send preOp", zap.String("deviceId", deviceSn), zap.String("pointName", op.PointName), zap.Any("value", op.Value))
					err = driverbox.WritePoint(deviceSn, plugin.PointData{
						PointName: op.PointName,
						Value:     op.Value,
					})
					if err != nil {
						return nil, err
					}
				}
			}

			err = bwc.transformData(ext.ObjType)
			if err != nil {
				return nil, err
			}
			if req, err := createWriteReq(bwc, ext); err == nil {
				req.DeviceId = deviceSn
				req.PointName = bwc.PointName
				writeReqs = append(writeReqs, req)
			} else {
				return nil, err
			}
		}
		return bacRequest{
			req:      writeReqs,
			mode:     mode,
			deviceId: deviceId,
		}, nil
	default:
		return nil, plugin.NotSupportEncode
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
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	rawJson := raw.(string)
	var resp readResponse
	err = json.Unmarshal([]byte(rawJson), &resp)
	if err != nil {
		return nil, err
	}
	// 当前设备未被lua解析
	pointDatalist := []plugin.PointData{{
		PointName: resp.PointName,
		Value:     resp.Value,
	}}
	res = append(res, plugin.DeviceData{
		ID:     resp.DeviceId,
		Values: pointDatalist,
	})

	return
}
