package iec104

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strconv"

	"github.com/orglibs/go-iecp5/asdu"
)

// builder 借用 fork 的标准编码器收集 ASDU，不创建连接，也不发送任何网络报文。
type builder struct {
	asdu.Connect
	params *asdu.Params // 地址宽度、源发地址和时区。
	frame  *asdu.ASDU   // 从编码器复制的报文，不引用临时发送缓冲区。
}

func (b *builder) Params() *asdu.Params { return b.params }
func (b *builder) Send(a *asdu.ASDU) error {
	b.frame = a.Clone()
	b.frame.OrigAddr = b.params.OrigAddress
	return nil
}
func (b *builder) UnderlyingConn() net.Conn { return nil }

// Command 校验值域后生成控制报文，不触发发送。bool/0/1 可用于单命令；
// 双命令仅接受 1/2；归一化设点、int16 设点、float32 设点分别检查范围。
// selectOnly 控制选择位，确认匹配与后续执行由主站连接器负责。
func Command(params *asdu.Params, p Point, value any, selectOnly bool) (*asdu.ASDU, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if params == nil {
		return nil, fmt.Errorf("missing ASDU parameters")
	}
	n, err := number(value)
	if err != nil {
		return nil, err
	}
	b := &builder{params: params}
	a := p.WriteAddress()
	ioa, ca := asdu.InfoObjAddr(a.IOA), asdu.CommonAddr(a.CommonAddress)
	cot := asdu.CauseOfTransmission{Cause: asdu.Activation}
	qoc := asdu.QualifierOfCommand{InSelect: selectOnly, Qual: asdu.QOCQual(p.Qualifier)}
	qos := asdu.QualifierOfSetpointCmd{InSelect: selectOnly, Qual: asdu.QOSQual(p.Qualifier)}
	switch asdu.TypeID(p.CommandType) {
	case asdu.C_SC_NA_1:
		if n != 0 && n != 1 {
			return nil, fmt.Errorf("single command requires 0 or 1")
		}
		err = asdu.SingleCmd(b, asdu.C_SC_NA_1, cot, ca, false, asdu.SingleCommandInfo{Ioa: ioa, Value: n == 1, Qoc: qoc})
	case asdu.C_DC_NA_1:
		if n != 1 && n != 2 {
			return nil, fmt.Errorf("double command requires 1 (off) or 2 (on)")
		}
		err = asdu.DoubleCmd(b, asdu.C_DC_NA_1, cot, ca, false, asdu.DoubleCommandInfo{Ioa: ioa, Value: asdu.DoubleCommand(n), Qoc: qoc})
	case asdu.C_SE_NA_1:
		if n < -1 || n > 32767.0/32768 {
			return nil, fmt.Errorf("normalized setpoint out of range")
		}
		err = asdu.SetpointCmdNormal(b, asdu.C_SE_NA_1, cot, ca, false, asdu.SetpointCommandNormalInfo{Ioa: ioa, Value: asdu.Normalize(math.Round(n * 32768)), Qos: qos})
	case asdu.C_SE_NB_1:
		if n < -32768 || n > 32767 || math.Trunc(n) != n {
			return nil, fmt.Errorf("scaled setpoint requires int16")
		}
		err = asdu.SetpointCmdScaled(b, asdu.C_SE_NB_1, cot, ca, false, asdu.SetpointCommandScaledInfo{Ioa: ioa, Value: int16(n), Qos: qos})
	case asdu.C_SE_NC_1:
		if math.Abs(n) > math.MaxFloat32 {
			return nil, fmt.Errorf("float setpoint out of range")
		}
		err = asdu.SetpointCmdFloat(b, asdu.C_SE_NC_1, cot, ca, false, asdu.SetpointCommandFloatInfo{Ioa: ioa, Value: float32(n), Qos: qos})
	default:
		return nil, fmt.Errorf("point has no supported commandType")
	}
	return b.frame, err
}

func number(v any) (float64, error) {
	var n float64
	var err error
	switch v := v.(type) {
	case bool:
		if v {
			n = 1
		}
	case int:
		n = float64(v)
	case int8:
		n = float64(v)
	case int16:
		n = float64(v)
	case int32:
		n = float64(v)
	case int64:
		n = float64(v)
	case uint:
		n = float64(v)
	case uint8:
		n = float64(v)
	case uint16:
		n = float64(v)
	case uint32:
		n = float64(v)
	case uint64:
		n = float64(v)
	case float32:
		n = float64(v)
	case float64:
		n = v
	case json.Number:
		n, err = v.Float64()
	case string:
		n, err = strconv.ParseFloat(v, 64)
	default:
		err = fmt.Errorf("unsupported command value %T", v)
	}
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, fmt.Errorf("invalid finite command value %v", v)
	}
	return n, nil
}
