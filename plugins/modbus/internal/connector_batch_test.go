package internal

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/internal/cache"
	exports "github.com/ibuilding-x/driver-box/v2/internal/export"
	"github.com/ibuilding-x/driver-box/v2/internal/logger"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/ibuilding-x/driver-box/v2/pkg/event"
	"github.com/ibuilding-x/driver-box/v2/pkg/luautil"
	"go.uber.org/zap"
)

type batchRecorder struct {
	mu      sync.Mutex
	calls   int
	batches []plugin.DeviceData
}

func (*batchRecorder) Init() error    { return nil }
func (*batchRecorder) Destroy() error { return nil }
func (*batchRecorder) IsReady() bool  { return true }
func (r *batchRecorder) OnEvent(code event.EventCode, _ string, _ interface{}) error {
	if code == event.DoExport {
		r.mu.Lock()
		r.calls++
		r.mu.Unlock()
	}
	return nil
}
func (r *batchRecorder) ExportTo(data plugin.DeviceData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	data.Values = append([]plugin.PointData(nil), data.Values...)
	r.batches = append(r.batches, data)
}
func (r *batchRecorder) take() (int, []plugin.DeviceData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	calls, batches := r.calls, r.batches
	r.calls, r.batches = 0, nil
	return calls, batches
}

// Exercise the real sendReadCommand -> driverbox.Export -> Shadow/change
// filter -> ExportTo path. The existing virtual transport avoids network I/O.
// Use a subprocess to isolate the framework's singleton cache, exporters,
// Shadow timers and asynchronous lifecycle events from other tests.
func TestSendReadCommandExportsReadGroupByDevice(t *testing.T) {
	if os.Getenv("DRIVERBOX_BATCH_TEST_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSendReadCommandExportsReadGroupByDevice$", "-test.timeout=20s")
		cmd.Env = append(os.Environ(), "DRIVERBOX_BATCH_TEST_CHILD=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("batch export integration test: %v\n%s", err, output)
		}
		return
	}
	logger.Logger = zap.NewNop()
	recorder := &batchRecorder{}
	exports.Exports = nil
	driverbox.EnableExport(recorder)
	config.ResourcePath = t.TempDir()
	points := []config.Point{
		{"name": "U", "description": "voltage", "valueType": "float", "reportMode": "change", "readWrite": "R"},
		{"name": "I", "description": "current", "valueType": "float", "reportMode": "change", "readWrite": "R"},
		{"name": "P", "description": "power", "valueType": "float", "reportMode": "change", "readWrite": "R"},
	}
	deviceConfig := config.DeviceConfig{
		PluginName: ProtocolName,
		DeviceModels: []config.DeviceModel{{
			Model:   config.Model{Name: "batch-meter", ModelID: "batch-meter", DevicePoints: points},
			Devices: []config.Device{{ID: "batch-A"}, {ID: "batch-B"}},
		}},
	}
	encoded, err := json.Marshal(deviceConfig)
	if err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(config.ResourcePath, "driver", ProtocolName)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.InitCoreCache(map[string]plugin.Plugin{ProtocolName: &Plugin{}}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"batch-A", "batch-B"} {
		driverbox.Shadow().AddDevice(id, "batch-meter")
	}
	script := filepath.Join(config.ResourcePath, "mock.lua")
	if err := os.WriteFile(script, []byte(`function mockRead(unit, kind, address, quantity)
  if address == 999 then error("read failed") end
  return "[220,10,100]"
end`), 0600); err != nil {
		t.Fatal(err)
	}
	ls, err = luautil.InitLuaVM(script)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()
	conn := &connector{virtual: true}
	makePoint := func(device, name string, address uint16) *Point {
		return &Point{Point: config.Point{"name": name}, DeviceId: device, Address: address,
			RegisterType: HoldingRegister, Quantity: 1, RawType: ValueTypeUint16}
	}
	group := &pointGroup{UnitID: 1, RegisterType: HoldingRegister, Quantity: 3, Points: []*Point{
		makePoint("batch-A", "U", 0),
		makePoint("batch-A", "I", 1),
		makePoint("batch-A", "P", 2),
	}}
	assertBatch := func(wantCalls int, want []plugin.DeviceData) {
		t.Helper()
		calls, got := recorder.take()
		if calls != wantCalls || !reflect.DeepEqual(got, want) {
			t.Fatalf("Export calls=%d, batches=%#v; want calls=%d, batches=%#v", calls, got, wantCalls, want)
		}
	}
	data := func(device string, values ...plugin.PointData) plugin.DeviceData {
		return plugin.DeviceData{ID: device, Values: values, ExportType: plugin.RealTimeExport}
	}
	voltage := plugin.PointData{PointName: "U", Value: 220.0}
	current := plugin.PointData{PointName: "I", Value: 10.0}
	power := plugin.PointData{PointName: "P", Value: 100.0}
	read := func() {
		t.Helper()
		if err := conn.sendReadCommand(group); err != nil {
			t.Fatal(err)
		}
	}
	read()
	assertBatch(1, []plugin.DeviceData{data("batch-A", voltage, current, power)})
	for _, value := range []plugin.PointData{voltage, current, power} {
		got, err := driverbox.Shadow().GetDevicePoint("batch-A", value.PointName)
		if err != nil || got != value.Value {
			t.Fatalf("Shadow %s=%v, %v", value.PointName, got, err)
		}
	}
	// An unchanged group still reaches Export once, but is filtered before ExportTo.
	read()
	assertBatch(1, nil)
	// Only changed points remain together in one callback.
	for _, name := range []string{"U", "P"} {
		if err := driverbox.Shadow().SetDevicePoint("batch-A", name, 0.0); err != nil {
			t.Fatal(err)
		}
	}
	read()
	assertBatch(1, []plugin.DeviceData{data("batch-A", voltage, power)})
	// Interleaved device points are grouped in first-seen order.
	_ = driverbox.Shadow().DeleteDevice("batch-A", "batch-B")
	driverbox.Shadow().AddDevice("batch-A", "batch-meter")
	driverbox.Shadow().AddDevice("batch-B", "batch-meter")
	group.Points[1].DeviceId = "batch-B"
	read()
	assertBatch(1, []plugin.DeviceData{data("batch-A", voltage, power), data("batch-B", current)})
	// Unsupported points do not discard other valid points in the read group.
	_ = driverbox.Shadow().SetDevicePoint("batch-A", "P", 0.0)
	group.Points[0].RawType = "unsupported"
	read()
	assertBatch(1, []plugin.DeviceData{data("batch-A", power)})
	for _, point := range group.Points {
		point.RawType = "unsupported"
	}
	read()
	assertBatch(0, nil)
	group.Points = nil
	read()
	assertBatch(0, nil)
	group.Address = 999
	if err := conn.sendReadCommand(group); err == nil {
		t.Fatal("expected read error")
	}
	assertBatch(0, nil)
	group.Quantity = 0
	if err := conn.sendReadCommand(group); err == nil {
		t.Fatal("expected zero quantity error")
	}
	assertBatch(0, nil)
}
