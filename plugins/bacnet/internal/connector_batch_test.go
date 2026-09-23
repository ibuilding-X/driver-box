package internal

import (
	"math"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/internal/testutil/exporttest"
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/btypes"
)

func TestReadResponseBatch(t *testing.T) {
	exporttest.Run(t, ProtocolName, &Plugin{}, func(r *exporttest.Recorder) {
		conn := &connector{}
		makeObject := func(instance uint32, device, point string, value interface{}) btypes.Object {
			return btypes.Object{ID: btypes.ObjectID{Instance: btypes.ObjectInstance(instance)}, Points: map[string]string{device: point}, Properties: []btypes.Property{{Type: btypes.PROP_PRESENT_VALUE, Data: value}}}
		}
		req := btypes.MultiplePropertyData{Objects: []btypes.Object{
			makeObject(1, "A", "U", 220.0), makeObject(2, "B", "I", 10.0), makeObject(3, "A", "P", 100.0),
		}}
		out := req
		if err := conn.exportReadResponse(req, out); err != nil {
			t.Fatal(err)
		}
		r.Assert(t, 1, exporttest.Data("A", plugin.PointData{PointName: "U", Value: 220.0}, plugin.PointData{PointName: "P", Value: 100.0}), exporttest.Data("B", plugin.PointData{PointName: "I", Value: 10.0}))
		if err := conn.exportReadResponse(req, out); err != nil {
			t.Fatal(err)
		}
		r.Assert(t, 1)
		// Invalid status, JSON encoding failure and an unmatched object must not lose valid values.
		out.Objects = []btypes.Object{
			{ID: req.Objects[0].ID, Properties: []btypes.Property{{Type: btypes.PROP_STATUS_FLAGS, Data: "invalid"}}},
			makeObject(2, "B", "I", math.NaN()), makeObject(3, "A", "P", 200.0), makeObject(4, "A", "U", 999.0),
		}
		if err := conn.exportReadResponse(req, out); err != nil {
			t.Fatal(err)
		}
		r.Assert(t, 1, exporttest.Data("A", plugin.PointData{PointName: "P", Value: 200.0}))
		out.Objects = out.Objects[:2]
		if err := conn.exportReadResponse(req, out); err != nil {
			t.Fatal(err)
		}
		r.Assert(t, 0)
		out.Objects = nil
		if err := conn.exportReadResponse(req, out); err != nil {
			t.Fatal(err)
		}
		r.Assert(t, 0)
		out.ErrorClass = 1
		if err := conn.exportReadResponse(req, out); err == nil {
			t.Fatal("expected read error")
		}
		r.Assert(t, 0)
	})
}
