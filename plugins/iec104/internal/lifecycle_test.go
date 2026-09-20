package internal

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
)

func TestDisconnectMarksEveryDeviceOfflineAndReconnectReports(t *testing.T) {
	offline := make(chan string, 16)
	c, frames, reports := startTestConnectorWithSetup(t, false, false, func(c *connector) {
		c.markOffline = func(id string) { offline <- id }
	})
	nextFrame(t, frames, 100)
	nextFrame(t, frames, 100)
	waitReports := func() {
		t.Helper()
		seen := map[string]bool{}
		deadline := time.After(3 * time.Second)
		for len(seen) < 3 {
			select {
			case d := <-reports:
				seen[d.ID] = true
			case <-deadline:
				t.Fatalf("missing reports: %v", seen)
			}
		}
	}
	waitReports()
	c.client.Disconnect()
	var ids []string
	for len(ids) < 3 {
		select {
		case id := <-offline:
			ids = append(ids, id)
		case <-time.After(3 * time.Second):
			t.Fatalf("missing offline notifications: %v", ids)
		}
	}
	sort.Strings(ids)
	if !reflect.DeepEqual(ids, []string{"a", "b", "c"}) {
		t.Fatalf("wrong disconnected devices: %v", ids)
	}
	nextFrame(t, frames, 100)
	nextFrame(t, frames, 100)
	waitReports()
}

type connectedForExport struct{ masterClient }

func (connectedForExport) IsConnected() bool { return true }

func TestDisconnectedSessionCannotExportQueuedSamples(t *testing.T) {
	s, cfg := fixture()
	var exported int
	c, err := newConnector(s, cfg, func([]plugin.DeviceData) { exported++ })
	if err != nil {
		t.Fatal(err)
	}
	defer c.cancel()
	c.client = connectedForExport{c.client}
	c.markOffline = func(string) {}
	batch := telemetryBatch{generation: 0, values: []plugin.DeviceData{{ID: "a"}}}
	c.exportBatch(batch)
	if exported != 1 {
		t.Fatal("current session did not export")
	}
	c.connectionLost()
	c.exportBatch(batch)
	if exported != 1 {
		t.Fatal("disconnected session exported queued data")
	}
	c.mu.Lock()
	c.generation++
	current := c.generation
	c.mu.Unlock()
	c.exportBatch(batch)
	if exported != 1 {
		t.Fatal("old data survived a reconnect")
	}
	batch.generation = current
	c.exportBatch(batch)
	if exported != 2 {
		t.Fatal("new session data was rejected")
	}
}

func TestOfflineTransitionFollowsInFlightExport(t *testing.T) {
	s, cfg := fixture()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	c, err := newConnector(s, cfg, func([]plugin.DeviceData) { close(entered); <-release })
	if err != nil {
		t.Fatal(err)
	}
	defer c.cancel()
	c.client = connectedForExport{c.client}
	offline := make(chan string, 3)
	c.markOffline = func(id string) { offline <- id }
	go c.exportBatch(telemetryBatch{generation: 0})
	<-entered
	go func() { c.connectionLost(); close(done) }()
	select {
	case <-offline:
		t.Fatal("offline notification overtook in-flight export")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("offline transition blocked")
	}
	if len(offline) != 3 {
		t.Fatalf("offline count=%d", len(offline))
	}
}
