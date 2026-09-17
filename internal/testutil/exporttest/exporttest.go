// Package exporttest runs export integration tests with isolated framework state.
package exporttest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sync"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/internal/cache"
	exports "github.com/ibuilding-x/driver-box/v2/internal/export"
	"github.com/ibuilding-x/driver-box/v2/internal/logger"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
	"github.com/ibuilding-x/driver-box/v2/pkg/event"
	"go.uber.org/zap"
)

type Recorder struct {
	mu      sync.Mutex
	calls   int
	batches []plugin.DeviceData
	events  map[event.EventCode][]interface{}
}

func (*Recorder) Init() error    { return nil }
func (*Recorder) Destroy() error { return nil }
func (*Recorder) IsReady() bool  { return true }
func (r *Recorder) OnEvent(code event.EventCode, _ string, value interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if code == event.DoExport {
		r.calls++
	}
	r.events[code] = append(r.events[code], value)
	return nil
}
func (r *Recorder) ExportTo(data plugin.DeviceData) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, data)
}
func (r *Recorder) Assert(t *testing.T, calls int, batches ...plugin.DeviceData) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.calls != calls || !reflect.DeepEqual(r.batches, batches) {
		t.Fatalf("Export calls=%d batches=%#v; want calls=%d batches=%#v", r.calls, r.batches, calls, batches)
	}
	r.calls, r.batches = 0, nil
}
func (r *Recorder) Events(code event.EventCode) []interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]interface{}(nil), r.events[code]...)
}

// Run isolates singleton caches, Shadow timers and asynchronous lifecycle events.
// The fixture provides devices A and B with float points U, I and P in change mode.
func Run(t *testing.T, protocol string, p plugin.Plugin, test func(*Recorder)) {
	t.Helper()
	if os.Getenv("DRIVERBOX_EXPORT_TEST_CHILD") != t.Name() {
		cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$", "-test.timeout=20s")
		cmd.Env = append(os.Environ(), "DRIVERBOX_EXPORT_TEST_CHILD="+t.Name())
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("export integration test: %v\n%s", err, output)
		}
		return
	}
	logger.Logger = zap.NewNop()
	r := &Recorder{events: make(map[event.EventCode][]interface{})}
	exports.Exports = nil
	driverbox.EnableExport(r)
	config.ResourcePath = t.TempDir()
	var points []config.Point
	for _, name := range []string{"U", "I", "P"} {
		points = append(points, config.Point{"name": name, "description": name, "valueType": "float", "reportMode": "change", "readWrite": "R"})
	}
	cfg := config.DeviceConfig{PluginName: protocol, DeviceModels: []config.DeviceModel{{
		Model:   config.Model{Name: "batch", ModelID: "batch", DevicePoints: points},
		Devices: []config.Device{{ID: "A"}, {ID: "B"}},
	}}}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(config.ResourcePath, "driver", protocol)
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), encoded, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.InitCoreCache(map[string]plugin.Plugin{protocol: p}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"A", "B"} {
		driverbox.Shadow().AddDevice(id, "batch")
	}
	test(r)
}

func Data(id string, points ...plugin.PointData) plugin.DeviceData {
	return plugin.DeviceData{ID: id, Values: points, ExportType: plugin.RealTimeExport}
}
