package internal

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/pkg/config"
)

func modelPoint() config.Point {
	return config.Point{"name": "switch", "readWrite": "RW", "ext": map[string]any{"ioa": 1, "typeId": 1, "commandType": 45, "selectBeforeExecute": true}}
}

func TestDeviceAddressOffsets(t *testing.T) {
	for _, tt := range []struct {
		name        string
		properties  map[string]string
		read, write uint32
		ca          uint16
		valid       bool
	}{
		{"defaults", nil, 1, 1, 1, true},
		{"shared model", map[string]string{"ioaOffset": "1000"}, 1001, 1001, 1, true},
		{"independent command", map[string]string{"ioaOffset": "1000", "commandIoaOffset": "2000", "commonAddress": "2"}, 1001, 2001, 2, true},
		{"hex", map[string]string{"ioaOffset": "0x100"}, 257, 257, 1, true},
		{"negative", map[string]string{"ioaOffset": "-1"}, 0, 0, 1, true},
		{"underflow", map[string]string{"ioaOffset": "-2"}, 0, 0, 0, false},
		{"overflow", map[string]string{"ioaOffset": "16777215"}, 0, 0, 0, false},
		{"huge offset", map[string]string{"ioaOffset": "9223372036854775807"}, 0, 0, 0, false},
		{"broadcast", map[string]string{"commonAddress": "65535"}, 0, 0, 0, false},
		{"bad offset", map[string]string{"ioaOffset": "one"}, 0, 0, 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			n, err := resolvePoint(modelPoint(), config.Device{ID: "dev", Properties: tt.properties}, 1)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, err)
			}
			if err == nil && (n.IOA != tt.read || n.WriteAddress().IOA != tt.write || n.CommonAddress != tt.ca) {
				t.Fatalf("bad mapping %+v write %+v", n, n.WriteAddress())
			}
		})
	}
	p := modelPoint()
	p["ext"].(map[string]any)["commandIoa"] = 500
	n, err := resolvePoint(p, config.Device{ID: "dev", Properties: map[string]string{"ioaOffset": "1000"}}, 1)
	if err != nil || n.IOA != 1001 || n.WriteAddress().IOA != 1500 {
		t.Fatalf("explicit command address: %+v %v", n, err)
	}
}

func fixture() (settings, config.DeviceConfig) {
	s, _ := parseSettings(map[string]any{"address": "127.0.0.1:2404"})
	s.ConnectionKey = "station"
	cfg := config.DeviceConfig{DeviceModels: []config.DeviceModel{{Model: config.Model{Name: "switch", DevicePoints: []config.Point{modelPoint()}}, Devices: []config.Device{
		{ID: "a", ConnectionKey: "station", Properties: map[string]string{"ioaOffset": "1000", "commandIoaOffset": "3000"}},
		{ID: "b", ConnectionKey: "station", Properties: map[string]string{"ioaOffset": "2000"}},
		{ID: "c", ConnectionKey: "station", Properties: map[string]string{"commonAddress": "2"}},
	}}}}
	return s, cfg
}

func TestAddressConflicts(t *testing.T) {
	s, cfg := fixture()
	c, err := newConnector(s, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.cancel()
	if len(c.nodes) != 3 || len(c.stations) != 2 {
		t.Fatalf("devices=%d stations=%v", len(c.nodes), c.stations)
	}
	cfg.DeviceModels[0].Devices[1].Properties["ioaOffset"] = "1000"
	if _, err = newConnector(s, cfg, nil); err == nil {
		t.Fatal("duplicate monitoring address accepted")
	}
	cfg.DeviceModels[0].Devices[1].Properties["ioaOffset"] = "2000"
	cfg.DeviceModels[0].Devices[1].Properties["commandIoaOffset"] = "3000"
	if _, err = newConnector(s, cfg, nil); err == nil {
		t.Fatal("duplicate command address accepted")
	}
}

func TestSettingsValidation(t *testing.T) {
	for _, raw := range []map[string]any{{"address": ""}, {"address": "udp://localhost:2404"}, {"address": "localhost:99999"}, {"address": "localhost", "commandTimeout": "0s"}, {"address": "localhost", "interrogationInterval": "-1s"}, {"address": "localhost", "connectTimeout": "256s"}, {"address": "localhost", "timeZone": "bad/timezone"}} {
		if _, err := parseSettings(raw); err == nil {
			t.Fatalf("invalid settings accepted: %v", raw)
		}
	}
	s, err := parseSettings(map[string]any{"address": "[::1]"})
	if err != nil || s.Address != "tcp://[::1]:2404" {
		t.Fatalf("IPv6: %+v %v", s, err)
	}
}

// 仓库示例使用真实配置解析路径，避免文档中的数字/字符串类型与实现脱节。
func TestRepositoryExample(t *testing.T) {
	raw, err := os.ReadFile("../../../res/driver/iec104/config.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.DeviceConfig
	if err = json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.PluginName != ProtocolName {
		t.Fatal("incorrect protocolName")
	}
	for key, raw := range cfg.Connections {
		s, err := parseSettings(raw)
		if err != nil {
			t.Fatal(err)
		}
		s.ConnectionKey = key
		if s.Enable {
			t.Fatal("example should not connect until explicitly configured")
		}
		c, err := newConnector(s, cfg, nil)
		if err != nil {
			t.Fatal(err)
		}
		c.cancel()
		if len(c.nodes) != 3 || c.nodes["cabinet_a"]["switch"].IOA != 1001 || c.nodes["cabinet_b"]["switch"].WriteAddress().IOA != 20001 || c.nodes["cabinet_c"]["voltage"].CommonAddress != 2 {
			t.Fatal("example address mapping incorrect")
		}
	}
	p := new(Plugin)
	p.Initialize(cfg)
	if _, err = p.Connector("cabinet_a"); err == nil {
		t.Fatal("disabled connection available")
	}
	if err = p.Destroy(); err != nil {
		t.Fatal(err)
	}
}
