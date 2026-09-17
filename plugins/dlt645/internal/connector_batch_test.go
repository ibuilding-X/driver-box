package internal

import (
	"testing"

	"github.com/ibuilding-x/driver-box/v2/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/v2/internal/testutil/exporttest"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
)

func TestReadGroupBatch(t *testing.T) {
	exporttest.Run(t, ProtocolName, &Plugin{}, func(r *exporttest.Recorder) {
		conn := &connector{virtual: true}
		group := &pointGroup{Points: []*Point{
			{Point: config.Point{"name": "U"}, DeviceId: "A"},
			{Point: config.Point{"name": "I"}, DeviceId: "B"},
			{Point: config.Point{"name": "P"}, DeviceId: "A"},
		}}
		if err := conn.sendReadCommand(group); err != nil {
			t.Fatal(err)
		}
		// The existing point conversion normalizes float zero to int zero.
		r.Assert(t, 1, exporttest.Data("A", plugin.PointData{PointName: "U", Value: 0}, plugin.PointData{PointName: "P", Value: 0}), exporttest.Data("B", plugin.PointData{PointName: "I", Value: 0}))
		if err := conn.sendReadCommand(group); err != nil {
			t.Fatal(err)
		}
		r.Assert(t, 1)
		group.Points = nil
		if err := conn.sendReadCommand(group); err != nil {
			t.Fatal(err)
		}
		r.Assert(t, 0)
	})
}
