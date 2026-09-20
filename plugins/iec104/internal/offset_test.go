package internal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/orglibs/go-iecp5/asdu"
)

// 独立列出全部已支持监视类型，确保 CP24/CP56 变体与无时标点选择相同地址区。
func TestMonitoringOffsetCategories(t *testing.T) {
	for _, group := range []struct {
		types []int
		want  uint32
	}{
		{[]int{1, 2, 30, 3, 4, 31, 7, 8, 33}, 1001},
		{[]int{5, 6, 32, 9, 10, 21, 34, 11, 12, 35, 13, 14, 36, 15, 16, 37}, 4001},
	} {
		for _, typ := range group.types {
			t.Run(fmt.Sprint(typ), func(t *testing.T) {
				p := config.Point{"name": "value", "readWrite": "R", "ext": map[string]any{"ioa": 1, "typeId": typ}}
				n, err := resolvePoint(p, config.Device{ID: "dev", Properties: map[string]string{"signalIoaOffset": "1000", "telemetryIoaOffset": "4000"}}, 1)
				if err != nil || n.IOA != group.want {
					t.Fatalf("wrong address: %+v, %v", n, err)
				}
			})
		}
	}
}

func TestControlOffsetCategories(t *testing.T) {
	for _, typ := range []int{45, 46, 48, 49, 50} {
		for _, explicitBase := range []bool{false, true} {
			p := modelPoint()
			ext := p["ext"].(map[string]any)
			ext["typeId"] = 13 // 即使监视点为遥测，单/双命令仍必须选遥控区。
			ext["commandType"] = typ
			base := uint32(1)
			if explicitBase {
				ext["commandIoa"] = 20
				base = 20
			}
			n, err := resolvePoint(p, config.Device{Properties: map[string]string{"signalIoaOffset": "1000", "telemetryIoaOffset": "4000", "commandIoaOffset": "10000", "setpointIoaOffset": "20000"}}, 1)
			want := base + 10000
			if typ >= 48 {
				want = base + 20000
			}
			if err != nil || n.IOA != 4001 || n.WriteAddress().IOA != want {
				t.Fatalf("type %d explicit=%v: %+v, %v", typ, explicitBase, n, err)
			}
			// 未配置遥调偏移时不继承遥测或遥控偏移。
			n, err = resolvePoint(p, config.Device{Properties: map[string]string{"telemetryIoaOffset": "4000"}}, 1)
			if err != nil || n.WriteAddress().IOA != base {
				t.Fatalf("unexpected offset inheritance: %+v, %v", n, err)
			}
		}
	}
}

func TestOffsetValidation(t *testing.T) {
	for _, key := range []string{"signalIoaOffset", "telemetryIoaOffset", "commandIoaOffset", "setpointIoaOffset"} {
		for _, value := range []string{"", "bad", "16777216", "-16777216", "9223372036854775807"} {
			if _, err := resolvePoint(modelPoint(), config.Device{Properties: map[string]string{key: value}}, 1); err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("%s=%q accepted or missing diagnostic: %v", key, value, err)
			}
		}
	}
	for _, value := range []string{"", "0", "1000"} {
		if _, err := resolvePoint(modelPoint(), config.Device{Properties: map[string]string{"ioaOffset": value}}, 1); err == nil {
			t.Fatal("obsolete ioaOffset accepted")
		}
	}
	for _, key := range []string{"telemetryIoaOffset", "setpointIoaOffset"} {
		p := config.Point{"name": "value", "readWrite": "RW", "ext": map[string]any{"ioa": 1, "typeId": 13, "commandType": 50}}
		for _, value := range []string{"-2", "16777215"} {
			if _, err := resolvePoint(p, config.Device{Properties: map[string]string{key: value}}, 1); err == nil {
				t.Fatalf("resolved address outside 24 bits accepted: %s=%s", key, value)
			}
		}
		for _, value := range []string{"-1", "0x100"} {
			if _, err := resolvePoint(p, config.Device{Properties: map[string]string{key: value}}, 1); err != nil {
				t.Fatalf("valid signed/hex offset rejected: %v", err)
			}
		}
	}
}

// 同模型内遥信和遥测复用基础 IOA=1，分别验证上报路由、读取、选择及执行实际编码地址。
func TestIndependentOffsetsRoutingAndEncoding(t *testing.T) {
	s, cfg := fixture()
	cfg.DeviceModels[0].DevicePoints = append(cfg.DeviceModels[0].DevicePoints,
		config.Point{"name": "voltage", "readWrite": "RW", "ext": map[string]any{"ioa": 1, "typeId": 13, "commandType": 50, "selectBeforeExecute": true}})
	for i := range cfg.DeviceModels[0].Devices {
		props := cfg.DeviceModels[0].Devices[i].Properties
		props["telemetryIoaOffset"] = fmt.Sprint(4000 + i*1000)
		props["setpointIoaOffset"] = fmt.Sprint(20000 + i*1000)
	}
	c, err := newConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.cancel()
	for i, dev := range []string{"a", "b", "c"} {
		ca := byte(1)
		if i == 2 {
			ca = 2
		}
		for _, point := range []struct {
			name        string
			kind        byte
			read, write uint32
			payload     []byte
		}{
			{"switch", 1, []uint32{1001, 2001, 1}[i], []uint32{3001, 2001, 1}[i], []byte{1}},
			{"voltage", 13, uint32(4001 + i*1000), uint32(20001 + i*1000), []byte{0, 0, 0x48, 0x41, 0}}, // 12.5 + QDS。
		} {
			for _, mode := range []plugin.EncodeMode{plugin.ReadMode, plugin.WriteMode} {
				req, err := c.Encode(dev, mode, plugin.PointData{PointName: point.name, Value: 1})
				if err != nil {
					t.Fatal(err)
				}
				want, count := point.read, 1
				if mode == plugin.WriteMode {
					want, count = point.write, 2
				}
				ops := req.(*request).operations
				if len(ops) != count {
					t.Fatal("selection/execute stages missing")
				}
				for _, op := range ops {
					raw, err := op.frame.MarshalBinary()
					if err != nil {
						t.Fatal(err)
					}
					ioa := uint32(raw[6]) | uint32(raw[7])<<8 | uint32(raw[8])<<16
					if ioa != want || raw[4] != ca {
						t.Fatalf("wrong wire address: %x want %d", raw, want)
					}
				}
			}
			raw := append([]byte{point.kind, 1, 3, 0, ca, 0, byte(point.read), byte(point.read >> 8), byte(point.read >> 16)}, point.payload...)
			a := asdu.NewEmptyASDU(asdu.ParamsWide)
			if err := a.UnmarshalBinary(raw); err != nil {
				t.Fatal(err)
			}
			if err := c.ASDUHandler(nil, a); err != nil {
				t.Fatal(err)
			}
			select {
			case batch := <-c.telemetry:
				data := batch.values
				if len(data) != 1 || data[0].ID != dev || len(data[0].Values) != 1 || data[0].Values[0].PointName != point.name {
					t.Fatalf("wrong route: %+v", data)
				}
			default:
				t.Fatal("missing report")
			}
		}
	}
	// 遥测区覆盖后产生冲突，应拒绝整条连接，不因遥信区不同而放行。
	cfg.DeviceModels[0].Devices[1].Properties["telemetryIoaOffset"] = "4000"
	if _, err := newConnector(s, cfg, nil); err == nil {
		t.Fatal("telemetry collision accepted")
	}
	cfg.DeviceModels[0].Devices[1].Properties["telemetryIoaOffset"] = "5000"
	cfg.DeviceModels[0].Devices[1].Properties["setpointIoaOffset"] = "20000"
	if _, err := newConnector(s, cfg, nil); err == nil {
		t.Fatal("setpoint collision accepted")
	}
}
