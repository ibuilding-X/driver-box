package iec104

import (
	"fmt"
	"math"
	"time"

	"github.com/orglibs/go-iecp5/asdu"
)

// Sample 保留协议层的原始质量、时间和计数器状态，不依赖业务数据结构。
// 当前主站仅将有效 Value 转换成 PointData；后续从站 export 可继续使用完整元数据。
type Sample struct {
	Address
	// TypeID 是接收报文的实际类型，保留是否携带 CP24/CP56 时标的信息。
	TypeID asdu.TypeID
	// Value 为解码后的值：单点为 0/1，双点为 0..3，归一化值还原为 [-1, 32767/32768]。
	Value any
	// Quality 为质量描述字；计数器的 IV 也映射到此处，不丢弃其他原始计数状态。
	Quality asdu.QualityDescriptor
	// Time 为报文携带的时间，无时标类型为零值；CP24 日期由 fork 按配置时区解释。
	Time time.Time
	// Counter 仅累计量类型非 nil，保留序号、进位、调整、无效等 BCR 标志。
	Counter *asdu.BinaryCounterReading
}

// Valid 排除 IV（无效）与 NT（非当前值）。其他质量位仍保留在 Sample 中。
func (s Sample) Valid() bool { return s.Quality&(asdu.QDSInvalid|asdu.QDSNotTopical) == 0 }

// Decode 校验信息体长度后才调用 fork 的类型解码器，防止截断报文触发越界。
// 支持 SQ=0 多地址和 SQ=1 连续地址；未知类型返回错误，不把控制确认当作遥测。
func Decode(a *asdu.ASDU) ([]Sample, error) {
	if a == nil || a.Params == nil {
		return nil, fmt.Errorf("missing ASDU parameters")
	}
	if err := a.Params.Valid(); err != nil {
		return nil, err
	}
	if MonitoringFamily(a.Type) == 0 {
		return nil, fmt.Errorf("unsupported monitoring ASDU %d", a.Type)
	}
	size, err := asdu.GetInfoObjSize(a.Type)
	if err != nil {
		return nil, err
	}
	n := int(a.Variable.Number)
	required := n * (size + a.InfoObjAddrSize)
	if a.Variable.IsSequence {
		required = a.InfoObjAddrSize + n*size
	}
	if n == 0 || len(a.InfoObj) != required {
		return nil, fmt.Errorf("invalid ASDU information object length")
	}
	// fork 的 Get* 方法会移动 InfoObj 切片；复制后解码，保留调用方报文可重用。
	a = a.Clone()
	out := make([]Sample, 0, n)
	add := func(ioa asdu.InfoObjAddr, value any, q asdu.QualityDescriptor, t time.Time) {
		out = append(out, Sample{Address: Address{uint16(a.CommonAddr), uint32(ioa)}, TypeID: a.Type, Value: value, Quality: q, Time: t})
	}
	switch MonitoringFamily(a.Type) {
	case asdu.M_SP_NA_1:
		for _, v := range a.GetSinglePoint() {
			value := 0
			if v.Value {
				value = 1
			}
			add(v.Ioa, value, v.Qds, v.Time)
		}
	case asdu.M_DP_NA_1:
		for _, v := range a.GetDoublePoint() {
			add(v.Ioa, int(v.Value), v.Qds, v.Time)
		}
	case asdu.M_ST_NA_1:
		for _, v := range a.GetStepPosition() {
			add(v.Ioa, v.Value.Val, v.Qds, v.Time)
		}
	case asdu.M_BO_NA_1:
		for _, v := range a.GetBitString32() {
			add(v.Ioa, v.Value, v.Qds, v.Time)
		}
	case asdu.M_ME_NA_1:
		for _, v := range a.GetMeasuredValueNormal() {
			add(v.Ioa, v.Value.Float64(), v.Qds, v.Time)
		}
	case asdu.M_ME_NB_1:
		for _, v := range a.GetMeasuredValueScaled() {
			add(v.Ioa, v.Value, v.Qds, v.Time)
		}
	case asdu.M_ME_NC_1:
		for _, v := range a.GetMeasuredValueFloat() {
			q := v.Qds
			if math.IsNaN(float64(v.Value)) || math.IsInf(float64(v.Value), 0) {
				q |= asdu.QDSInvalid
			}
			add(v.Ioa, float64(v.Value), q, v.Time)
		}
	case asdu.M_IT_NA_1:
		for _, v := range a.GetIntegratedTotals() {
			q := asdu.QDSGood
			if v.Value.IsInvalid {
				q = asdu.QDSInvalid
			}
			add(v.Ioa, v.Value.CounterReading, q, v.Time)
			counter := v.Value
			out[len(out)-1].Counter = &counter
		}
	}
	for _, v := range out {
		if v.IOA > 0xffffff {
			return nil, fmt.Errorf("sequence IOA overflow")
		}
	}
	return out, nil
}
