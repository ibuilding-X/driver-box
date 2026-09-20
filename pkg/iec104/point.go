// Package iec104 提供与主从站角色无关的 IEC104 地址、点位及 ASDU 编解码。
// 本包不依赖 driverbox、连接池或影子服务，后续从站 export 可直接复用。
package iec104

import (
	"fmt"

	"github.com/orglibs/go-iecp5/asdu"
)

// Address 的唯一性限定在一条传输连接内；业务设备 ID 不属于 IEC104 协议地址。
type Address struct {
	// CommonAddress 为 ASDU 公共地址 CA，有效范围 1..65534。
	CommonAddress uint16 `json:"commonAddress"`
	// IOA 为 24 位信息体地址。模型中填写相对地址，运行时 Point 中保存偏移后的绝对地址。
	IOA uint32 `json:"ioa"`
}

// Point 描述一个点的监视与控制信息，可用于主站采集和后续从站映射。
type Point struct {
	Address
	// TypeID 是监视方向类型标识，如 1=单点遥信、11=标度值、13=短浮点。
	// 路由时同一信息类型的带时标变体互通；不同信息类型不互相转换。
	TypeID uint8 `json:"typeId"`
	// CommandType 是控制方向类型标识；0 表示未配置写入，支持 45/46/48/49/50。
	CommandType uint8 `json:"commandType"`
	// CommandIOA 为控制地址；nil 表示沿用 IOA，指针可区分“未设置”和合法地址 0。
	// 最终控制地址 = 模型 commandIoa（未设时为原始 ioa）+ 设备对应控制偏移。
	// 45/46 使用 commandIoaOffset，48/49/50 使用 setpointIoaOffset，均默认 0。
	// 控制偏移不继承 signalIoaOffset 或 telemetryIoaOffset。
	CommandIOA *uint32 `json:"commandIoa,omitempty"`
	// SelectBeforeExecute=true 时先选择，收到匹配的正 ACT_CON 后才执行；默认直接执行。
	SelectBeforeExecute bool `json:"selectBeforeExecute"`
	// Qualifier 为 QOC/QOS 限定词。单/双命令可取 0..3，设点命令目前只接受 0。
	Qualifier uint8 `json:"qualifier"`
}

// Validate 检查地址、类型与限定词，避免未知类型或越界数值进入线上的编码器。
func (p Point) Validate() error {
	if p.CommonAddress == 0 || p.CommonAddress == 65535 {
		return fmt.Errorf("commonAddress must be in [1, 65534]")
	}
	if p.IOA > 0xffffff || (p.CommandIOA != nil && *p.CommandIOA > 0xffffff) {
		return fmt.Errorf("ioa must fit in 24 bits")
	}
	if MonitoringFamily(asdu.TypeID(p.TypeID)) == 0 {
		return fmt.Errorf("unsupported monitoring typeId %d", p.TypeID)
	}
	switch asdu.TypeID(p.CommandType) {
	case 0:
		if p.SelectBeforeExecute || p.CommandIOA != nil || p.Qualifier != 0 {
			return fmt.Errorf("command options require commandType")
		}
	case asdu.C_SC_NA_1, asdu.C_DC_NA_1:
		if p.Qualifier > 3 {
			return fmt.Errorf("command qualifier must be in [0, 3]")
		}
	case asdu.C_SE_NA_1, asdu.C_SE_NB_1, asdu.C_SE_NC_1:
		if p.Qualifier != 0 {
			return fmt.Errorf("setpoint qualifier must be 0")
		}
	default:
		return fmt.Errorf("unsupported commandType %d", p.CommandType)
	}
	return nil
}

// WriteAddress 返回控制方向的地址；不会修改原始监视地址。
func (p Point) WriteAddress() Address {
	a := p.Address
	if p.CommandIOA != nil {
		a.IOA = *p.CommandIOA
	}
	return a
}

// MonitoringFamily 将无时标总召和带时标自发上报归并到同一类型族。
// 不支持的类型返回 0；例如单点与双点绝不会被归到同一族。
func MonitoringFamily(t asdu.TypeID) asdu.TypeID {
	switch t {
	case asdu.M_SP_NA_1, asdu.M_SP_TA_1, asdu.M_SP_TB_1:
		return asdu.M_SP_NA_1
	case asdu.M_DP_NA_1, asdu.M_DP_TA_1, asdu.M_DP_TB_1:
		return asdu.M_DP_NA_1
	case asdu.M_ST_NA_1, asdu.M_ST_TA_1, asdu.M_ST_TB_1:
		return asdu.M_ST_NA_1
	case asdu.M_BO_NA_1, asdu.M_BO_TA_1, asdu.M_BO_TB_1:
		return asdu.M_BO_NA_1
	case asdu.M_ME_NA_1, asdu.M_ME_TA_1, asdu.M_ME_TD_1, asdu.M_ME_ND_1:
		return asdu.M_ME_NA_1
	case asdu.M_ME_NB_1, asdu.M_ME_TB_1, asdu.M_ME_TE_1:
		return asdu.M_ME_NB_1
	case asdu.M_ME_NC_1, asdu.M_ME_TC_1, asdu.M_ME_TF_1:
		return asdu.M_ME_NC_1
	case asdu.M_IT_NA_1, asdu.M_IT_TA_1, asdu.M_IT_TB_1:
		return asdu.M_IT_NA_1
	default:
		return 0
	}
}
