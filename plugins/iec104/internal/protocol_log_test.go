package internal

import (
	"bytes"
	"context"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	projectlog "github.com/ibuilding-x/driver-box/v2/internal/logger"
	"github.com/orglibs/go-iecp5/cs104"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestProtocolLogSummaryAndMalformedFrames(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	s, _ := fixture()
	if trace := newProtocolTrace(s, nil); trace != nil {
		t.Fatal("disabled logging installed observer")
	}
	s.ProtocolLogEnabled = true
	trace := newProtocolTrace(s, zap.New(core))
	// SQ=1 的两点上报只记录起始 IOA；完整 hex 保留其余信息体。
	raw, err := cs104.NewIFrame(4, 2, []byte{1, 0x82, 3, 7, 2, 0, 0xe9, 3, 0, 1, 0})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := append([]byte(nil), raw...)
	trace(false, raw)
	if !bytes.Equal(snapshot, raw) {
		t.Fatal("logging changed business frame")
	}
	fields := logs.All()[0].ContextMap()
	if fields["direction"] != "RX" || fields["connection"] != "station" || fields["apci"] != "I[sendNO: 4, recvNO: 2]" || fields["ca"] != uint16(2) || fields["oa"] != uint8(7) || fields["firstIoa"] != uint32(1001) || fields["count"] != uint8(2) || fields["sq"] != true {
		t.Fatalf("bad summary: %#v", fields)
	}
	trace(true, cs104.NewUFrame(cs104.UStartDtActive))
	if fields := logs.All()[1].ContextMap(); fields["direction"] != "TX" || fields["hex"] != "68 04 07 00 00 00" {
		t.Fatalf("bad U frame: %#v", fields)
	}
	trace(false, []byte{0x68, 4, 7, 1, 0, 0}) // 非法 APCI 仍需可排查。
	badASDU, _ := cs104.NewIFrame(0, 0, []byte{45, 1, 6, 0, 1, 0, 1, 0, 0, 1, 0xaa})
	trace(false, badASDU) // 尾部多余字节不能被日志解析器截断隐藏。
	for _, entry := range logs.All()[2:] {
		if entry.ContextMap()["parseError"] == nil || entry.ContextMap()["hex"] == "" {
			t.Fatalf("missing malformed frame details: %#v", entry.ContextMap())
		}
	}
}

// 通过真实 connector 收发总召、遥测、选择/执行确认，验证配置接线以及连接间的开关隔离。
// 不只调用格式化函数；同时确认日志解析没有消费业务数据、没有影响多设备上报。
func TestProtocolLogConnectionIsolation(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	previous := projectlog.Logger
	projectlog.Logger = zap.New(core)
	defer func() { projectlog.Logger = previous }()
	for _, enabled := range []bool{true, false} {
		s, cfg := fixture()
		parsed, err := parseSettings(map[string]any{"address": "127.0.0.1:2404", "protocolLogEnabled": enabled})
		if err != nil {
			t.Fatal(err)
		}
		s.ProtocolLogEnabled = parsed.ProtocolLogEnabled
		if !enabled {
			s.ConnectionKey = "quiet"
			for i := range cfg.DeviceModels[0].Devices {
				cfg.DeviceModels[0].Devices[i].ConnectionKey = s.ConnectionKey
			}
		}
		peerDone := make(chan struct{})
		s.dialContext = func(context.Context, *url.URL) (net.Conn, error) {
			client, server := net.Pipe()
			go func() {
				defer close(peerDone)
				(&testStation{conn: server, observed: make(chan []byte, 64)}).run()
			}()
			return client, nil
		}
		reports := make(chan plugin.DeviceData, 16)
		c, err := newConnector(s, cfg, func(data []plugin.DeviceData) {
			for _, d := range data {
				reports <- d
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		func() {
			if err := c.start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close(); <-peerDone }()
			devices := map[string]bool{}
			for len(devices) < 3 {
				select {
				case report := <-reports:
					devices[report.ID] = true
				case <-time.After(3 * time.Second):
					t.Fatal("logging interfered with telemetry")
				}
			}
			req, err := c.Encode("a", plugin.WriteMode, plugin.PointData{PointName: "switch", Value: 1})
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Send(req); err != nil {
				t.Fatalf("logging interfered with command confirmation: %v", err)
			}
		}()
	}
	seen := map[string]bool{}
	for _, entry := range logs.All() {
		fields := entry.ContextMap()
		if fields["connection"] != "station" {
			t.Fatalf("disabled connection produced protocol log: %#v", fields)
		}
		apci, _ := fields["apci"].(string)
		if apci != "" {
			seen[fields["direction"].(string)+apci[:1]] = true
		}
		if fields["typeId"] == uint8(45) && fields["direction"] == "RX" && strings.Contains(fields["hex"].(string), "B9 0B 00 81") {
			seen["selectConfirmation"] = true
		}
	}
	for _, key := range []string{"TXU", "RXU", "TXI", "RXI", "RXS", "selectConfirmation"} {
		if !seen[key] {
			t.Errorf("missing real transport log %s", key)
		}
	}
}
