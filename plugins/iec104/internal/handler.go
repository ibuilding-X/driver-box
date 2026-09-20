package internal

import (
	"bytes"
	"fmt"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	protocol "github.com/ibuilding-x/driver-box/v2/pkg/iec104"
	"github.com/orglibs/go-iecp5/asdu"
)

// ASDUHandler 将监视方向报文路由到实际设备。测试报文、无效值和未知点不更新影子；
// 控制确认仅交给 confirm，绝不拿写入回显替代设备实际测量值。
func (c *connector) ASDUHandler(_ asdu.Connect, a *asdu.ASDU) error {
	if c.ctx.Err() != nil || a.Coa.IsTest {
		return nil
	}
	if protocol.MonitoringFamily(a.Type) == 0 {
		return c.confirm(a)
	}
	if a.Coa.IsNegative {
		return fmt.Errorf("negative monitoring ASDU: %s", a.Coa)
	}
	samples, err := protocol.Decode(a)
	if err != nil {
		return err
	}
	data := make([]plugin.DeviceData, 0)
	devices := make(map[string]int)
	for _, sample := range samples {
		n, ok := c.routes[route{sample.Address, protocol.MonitoringFamily(sample.TypeID)}]
		if !ok || !sample.Valid() {
			continue
		}
		index, ok := devices[n.deviceID]
		if !ok {
			index = len(data)
			devices[n.deviceID] = index
			data = append(data, plugin.DeviceData{ID: n.deviceID, ExportType: plugin.RealTimeExport})
		}
		data[index].Values = append(data[index].Values, plugin.PointData{PointName: n.name, Value: sample.Value})
	}
	if len(data) > 0 {
		select {
		case c.telemetry <- data:
		case <-c.ctx.Done():
		}
	}
	return nil
}

// confirm 精确匹配请求体（包含 IOA、值和选择位），只接受正 ACT_CON。
// ACT_TERM 不用于提前判定成功；负确认或未知类型/原因/地址响应传回调用方。
func (c *connector) confirm(a *asdu.ASDU) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	pending := c.pending
	if pending == nil {
		return nil
	}
	expected := pending.frame
	if a.Type != expected.Type || a.CommonAddr != expected.CommonAddr || a.OrigAddr != expected.OrigAddr || a.Variable != expected.Variable || !bytes.Equal(a.InfoObj, expected.InfoObj) {
		return nil
	}
	var result error
	if a.Coa.IsNegative || a.Coa.Cause >= asdu.UnknownTypeID && a.Coa.Cause <= asdu.UnknownIOA {
		result = fmt.Errorf("IEC104 command rejected: %s", a.Coa)
	} else if a.Coa.Cause != asdu.ActivationCon {
		return nil
	}
	select {
	case pending.result <- result:
	default:
	}
	return nil
}

func systemResponse(a *asdu.ASDU) error {
	if a.Coa.IsNegative || a.Coa.Cause >= asdu.UnknownTypeID && a.Coa.Cause <= asdu.UnknownIOA {
		return fmt.Errorf("IEC104 system command rejected: %s", a.Coa)
	}
	return nil
}
func (c *connector) InterrogationHandler(_ asdu.Connect, a *asdu.ASDU) error {
	return systemResponse(a)
}
func (c *connector) CounterInterrogationHandler(_ asdu.Connect, a *asdu.ASDU) error {
	return systemResponse(a)
}
func (c *connector) ReadHandler(_ asdu.Connect, a *asdu.ASDU) error         { return systemResponse(a) }
func (c *connector) TestCommandHandler(_ asdu.Connect, a *asdu.ASDU) error  { return systemResponse(a) }
func (c *connector) ClockSyncHandler(_ asdu.Connect, a *asdu.ASDU) error    { return systemResponse(a) }
func (c *connector) ResetProcessHandler(_ asdu.Connect, a *asdu.ASDU) error { return systemResponse(a) }
func (c *connector) DelayAcquisitionHandler(_ asdu.Connect, a *asdu.ASDU) error {
	return systemResponse(a)
}
