package plugin

import (
	"reflect"
	"testing"

	"github.com/ibuilding-x/driver-box/v2/pkg/event"
)

func TestMergeDeviceData(t *testing.T) {
	first := DeviceData{ID: "A", Values: []PointData{{PointName: "U", Value: 1}}, Events: []event.Data{{Code: "first"}}}
	input := []DeviceData{
		{ID: "empty"}, first,
		{ID: "B", Values: []PointData{{PointName: "I", Value: 2}}},
		{ID: "A", Values: []PointData{{PointName: "P", Value: 3}, {PointName: "U", Value: 4}}, Events: []event.Data{{Code: "second"}}},
		{ID: "A", ExportType: RealTimeExport, Values: []PointData{{PointName: "I", Value: 5}}},
		{ID: "event-only", Events: []event.Data{{Code: "alarm"}}},
	}
	got := MergeDeviceData(input)
	want := []DeviceData{
		{ID: "A", Values: []PointData{{PointName: "U", Value: 1}, {PointName: "P", Value: 3}, {PointName: "U", Value: 4}}, Events: []event.Data{{Code: "first"}, {Code: "second"}}},
		input[2], input[4], input[5],
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	got[0].Values[0].Value = 99
	got[0].Events[0].Code = "changed"
	if first.Values[0].Value != 1 || first.Events[0].Code != "first" {
		t.Fatal("batch aliases input slices")
	}
	if got := MergeDeviceData(nil); len(got) != 0 {
		t.Fatalf("nil input: %#v", got)
	}
	if got := MergeDeviceData([]DeviceData{{ID: "empty"}}); len(got) != 0 {
		t.Fatalf("empty data: %#v", got)
	}
}
