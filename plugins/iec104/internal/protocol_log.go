package internal

import (
	"fmt"

	"github.com/orglibs/go-iecp5/asdu"
	"github.com/orglibs/go-iecp5/cs104"
	"go.uber.org/zap"
)

// newProtocolTrace 只为开启协议日志的连接安装观察器，避免关闭时复制/格式化报文。
// 使用项目的并发安全日志器；Info 级使默认日志配置即可查看报文，无需打开全局 Debug。
// 一帧可携带多个设备的数据，故固定标注连接身份，CA/IOA 使用线上实际地址，不猜测设备 ID。
func newProtocolTrace(s settings, logger *zap.Logger) func(bool, []byte) {
	if !s.ProtocolLogEnabled {
		return nil
	}
	log := logger.With(zap.String("protocol", ProtocolName), zap.String("connection", s.ConnectionKey), zap.String("address", s.Address))
	return func(outbound bool, raw []byte) {
		direction := "RX"
		if outbound {
			direction = "TX"
		}
		// 尊重项目日志级别；若全局过滤 Info，跳过十六进制格式化和摘要解码。
		entry := log.Check(zap.InfoLevel, "IEC104 protocol frame")
		if entry == nil {
			return
		}
		fields := []zap.Field{zap.String("direction", direction), zap.Int("bytes", len(raw)), zap.String("hex", fmt.Sprintf("% X", raw))}
		header, payload, err := cs104.ParseChecked(raw)
		if err != nil {
			entry.Write(append(fields, zap.String("parseError", err.Error()))...)
			return
		}
		fields = append(fields, zap.String("apci", fmt.Sprint(header)))
		if _, ok := header.(cs104.IAPCI); ok {
			// 在独立 ASDU 上解析，日志不会消费业务处理器的 InfoObj，也不改变质量/时标。
			a := asdu.NewEmptyASDU(&s.params)
			if err := a.UnmarshalBinary(payload); err != nil {
				fields = append(fields, zap.String("parseError", err.Error()))
			} else {
				fields = append(fields,
					zap.Uint8("typeId", uint8(a.Type)), zap.String("type", a.Type.String()),
					zap.Uint8("cot", uint8(a.Coa.Cause)), zap.Bool("negative", a.Coa.IsNegative), zap.Bool("test", a.Coa.IsTest),
					zap.Uint8("oa", uint8(a.OrigAddr)), zap.Uint16("ca", uint16(a.CommonAddr)),
					zap.Uint8("count", a.Variable.Number), zap.Bool("sq", a.Variable.IsSequence))
				// firstIoa 仅代表第一个信息对象；SQ=0 的其余地址须结合完整 hex 查看。
				if len(a.InfoObj) >= a.InfoObjAddrSize {
					fields = append(fields, zap.Uint32("firstIoa", uint32(a.ReadInfoObjAddr())))
				}
			}
		}
		entry.Write(fields...)
	}
}
