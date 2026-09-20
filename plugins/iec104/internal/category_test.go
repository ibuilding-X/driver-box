package internal

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	protocol "github.com/ibuilding-x/driver-box/v2/pkg/iec104"
	"github.com/orglibs/go-iecp5/asdu"
)

func categoryFixture() (settings, config.DeviceConfig) {
	s, cfg := fixture()
	cfg.DeviceModels[0].Devices = cfg.DeviceModels[0].Devices[:1]
	cfg.DeviceModels[0].Devices[0].Properties["telemetryIoaOffset"] = "4000"
	cfg.DeviceModels[0].DevicePoints = append(cfg.DeviceModels[0].DevicePoints,
		config.Point{"name": "value", "readWrite": "R", "ext": map[string]any{"ioa": 1, "category": "telemetry"}})
	return s, cfg
}

func TestCategoryConfigurationAndUniqueAddress(t *testing.T) {
	for _, category := range []any{nil, "", "analog", 13} {
		p := modelPoint()
		p["ext"].(map[string]any)["category"] = category
		if _, err := resolvePoint(p, config.Device{}, 1); err == nil {
			t.Fatalf("invalid category accepted: %v", category)
		}
	}
	p := modelPoint()
	delete(p["ext"].(map[string]any), "category")
	if _, err := resolvePoint(p, config.Device{}, 1); err == nil {
		t.Fatal("missing category accepted")
	}
	for _, legacy := range []any{nil, 0, 1, 13} {
		p := modelPoint()
		p["ext"].(map[string]any)["typeId"] = legacy
		if _, err := resolvePoint(p, config.Device{}, 1); err == nil || !strings.Contains(err.Error(), "typeId") {
			t.Fatalf("obsolete typeId accepted: %v", err)
		}
	}
	s, cfg := categoryFixture()
	// 遥信/遥测即使类别不同，也不能在同连接同 CA 下占用相同的最终 IOA。
	cfg.DeviceModels[0].Devices[0].Properties["telemetryIoaOffset"] = "1000"
	if _, err := newConnector(s, cfg, nil); err == nil || !strings.Contains(err.Error(), "monitoring address conflict") {
		t.Fatalf("ambiguous address accepted: %v", err)
	}
}

// 各监视编码都按实际报文解码。给同一点连续发送不同编码，保留数值含义，
// 例如双点的 2 不改为 bool，标度的 -2 不按归一化值重新缩放。
func TestMonitoringCategoriesAcceptActualWireTypes(t *testing.T) {
	s, cfg := categoryFixture()
	c, err := newConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.cancel()
	for _, group := range []struct {
		types    []byte
		payload  []byte
		want     any
		category protocol.MonitoringCategory
	}{
		{[]byte{1, 2, 30}, []byte{1}, int(1), protocol.Signal},
		{[]byte{3, 4, 31}, []byte{2}, int(2), protocol.Signal},
		{[]byte{7, 8, 33}, []byte{42, 0, 0, 0, 0}, uint32(42), protocol.Signal},
		{[]byte{5, 6, 32}, []byte{0x7f, 0}, int(-1), protocol.Telemetry},
		{[]byte{9, 10, 34}, []byte{0, 0x40, 0}, float64(.5), protocol.Telemetry},
		{[]byte{21}, []byte{0, 0x40}, float64(.5), protocol.Telemetry},
		{[]byte{11, 12, 35}, []byte{0xfe, 0xff, 0}, int16(-2), protocol.Telemetry},
		{[]byte{13, 14, 36}, []byte{0, 0, 0xc0, 0x3f, 0}, float64(1.5), protocol.Telemetry},
		{[]byte{15, 16, 37}, []byte{42, 0, 0, 0, 0}, int32(42), protocol.Telemetry},
	} {
		for index, kind := range group.types {
			t.Run(fmt.Sprint(kind), func(t *testing.T) {
				ioa, name := uint32(1001), "switch"
				if group.category == protocol.Telemetry {
					ioa, name = 4001, "value"
				}
				payload := append([]byte(nil), group.payload...)
				if index == 1 {
					payload = append(payload, asdu.CP24Time2a(time.Now(), time.UTC)...)
				}
				if index == 2 {
					payload = append(payload, asdu.CP56Time2a(time.Now(), time.UTC, true)...)
				}
				if got := protocol.CategoryOf(asdu.TypeID(kind)); got != group.category {
					t.Fatalf("category %q", got)
				}
				for _, wrongCategory := range []bool{false, true} {
					address := ioa
					if wrongCategory {
						if address == 1001 {
							address = 4001
						} else {
							address = 1001
						}
					}
					raw := append([]byte{kind, 1, 3, 0, 1, 0, byte(address), byte(address >> 8), byte(address >> 16)}, payload...)
					a := asdu.NewEmptyASDU(asdu.ParamsWide)
					if err := a.UnmarshalBinary(raw); err != nil {
						t.Fatal(err)
					}
					if err := c.ASDUHandler(nil, a); err != nil {
						t.Fatal(err)
					}
					select {
					case batch := <-c.telemetry:
						if wrongCategory {
							t.Fatal("cross-category report accepted")
						}
						if len(batch.values) != 1 || batch.values[0].ID != "a" || len(batch.values[0].Values) != 1 || batch.values[0].Values[0].PointName != name || batch.values[0].Values[0].Value != group.want {
							t.Fatalf("wrong decoded report: %+v", batch.values)
						}
					default:
						if !wrongCategory {
							t.Fatal("supported encoding was filtered")
						}
					}
				}
			})
		}
	}
	if protocol.CategoryOf(200) != "" || protocol.CategoryOf(asdu.C_SC_NA_1) != "" {
		t.Fatal("unknown/control type classified as monitoring")
	}
}

// 跑完整客户端状态机，验证一个总召可按不同遥测编码更新同一模型点，
// 同时确认读取帧仅携带 CA/IOA，不依赖模型指定的预期 TypeID。
func TestMasterReadsWithoutExpectedType(t *testing.T) {
	s, cfg := categoryFixture()
	frames := make(chan []byte, 16)
	done := make(chan struct{})
	s.dialContext = func(context.Context, *url.URL) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer close(done)
			(&testStation{conn: server, observed: frames, monitoring: [][]byte{
				{9, 1, 20, 0, 1, 0, 0xa1, 0x0f, 0, 0, 0x40, 0},
				{11, 1, 20, 0, 1, 0, 0xa1, 0x0f, 0, 0xfe, 0xff, 0},
				{13, 1, 20, 0, 1, 0, 0xa1, 0x0f, 0, 0, 0, 0xc0, 0x3f, 0},
				{3, 1, 20, 0, 1, 0, 0xe9, 3, 0, 2},
			}}).run()
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
	c.markOffline = func(string) {}
	if err := c.start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close(); <-done }()
	for _, want := range []any{float64(.5), int16(-2), float64(1.5), int(2)} {
		select {
		case report := <-reports:
			if report.ID != "a" || len(report.Values) != 1 || report.Values[0].Value != want {
				t.Fatalf("wrong report: %+v want %v", report, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("supported wire type failed to reach export")
		}
	}
	req, err := c.Encode("a", plugin.ReadMode, plugin.PointData{PointName: "value"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Send(req); err != nil {
		t.Fatal(err)
	}
	raw := nextFrame(t, frames, 102)
	if len(raw) != 9 || raw[4] != 1 || raw[6] != 0xa1 || raw[7] != 0x0f || raw[8] != 0 {
		t.Fatalf("wrong read request: %x", raw)
	}
}
