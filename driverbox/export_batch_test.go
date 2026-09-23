package driverbox_test

import (
	"reflect"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/internal/testutil/exporttest"
	"github.com/ibuilding-x/driver-box/v2/pkg/event"
)

// No protocol methods are needed when loading the fixture configuration.
type fixturePlugin struct{ plugin.Plugin }

func TestExportBatch(t *testing.T) {
	exporttest.Run(t, "batch", &fixturePlugin{}, func(r *exporttest.Recorder) {
		u := plugin.PointData{PointName: "U", Value: 220.0}
		i := plugin.PointData{PointName: "I", Value: 10.0}
		p := plugin.PointData{PointName: "P", Value: 100.0}
		input := []plugin.DeviceData{exporttest.Data("A", u), exporttest.Data("B", i), exporttest.Data("A", p)}
		input[2].Events = []event.Data{{Code: "batch-alarm", Value: 1}}
		driverbox.Export(input)
		a := exporttest.Data("A", u, p)
		a.Events = input[2].Events
		r.Assert(t, 1, a, exporttest.Data("B", i))
		raw := r.Events(event.DoExport)
		if len(raw) != 1 || !reflect.DeepEqual(raw[0], []plugin.DeviceData{a, exporttest.Data("B", i)}) {
			t.Fatalf("raw batch: %#v", raw)
		}
		if got := r.Events("batch-alarm"); !reflect.DeepEqual(got, []interface{}{1}) {
			t.Fatalf("alarm events: %#v", got)
		}
		if got := r.Events(event.Exporting); len(got) != 2 {
			t.Fatalf("pre-export events: %#v", got)
		}
		driverbox.Export(input)
		r.Assert(t, 1)
		input[0].Values[0].Value = 230.0
		driverbox.Export(input)
		changed := exporttest.Data("A", plugin.PointData{PointName: "U", Value: 230.0})
		changed.Events = input[2].Events
		r.Assert(t, 1, changed)
		// Separate calls remain separate batches, even for the same device.
		driverbox.Export([]plugin.DeviceData{exporttest.Data("A", plugin.PointData{PointName: "I", Value: 20.0})})
		driverbox.Export([]plugin.DeviceData{exporttest.Data("A", plugin.PointData{PointName: "P", Value: 200.0})})
		r.Assert(t, 2, exporttest.Data("A", plugin.PointData{PointName: "I", Value: 20.0}), exporttest.Data("A", plugin.PointData{PointName: "P", Value: 200.0}))
		// Events survive a batch with no point values, without an ExportTo callback.
		driverbox.Export([]plugin.DeviceData{{ID: "A", Events: []event.Data{{Code: "event-only", Value: true}}}})
		r.Assert(t, 1)
		if got := r.Events("event-only"); !reflect.DeepEqual(got, []interface{}{true}) {
			t.Fatalf("event-only batch: %#v", got)
		}
		driverbox.Export(nil)
		driverbox.Export([]plugin.DeviceData{{ID: "A"}})
		r.Assert(t, 0)
	})
}
