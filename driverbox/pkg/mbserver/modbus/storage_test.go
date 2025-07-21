package modbus

import (
	"testing"
)

func TestStorage(t *testing.T) {
	cs := &coilStorage{}
	t.Run("coilStorage", func(t *testing.T) {
		err := cs.Write(100, []bool{true, false})
		if err != nil {
			t.Fatalf("coilStorage write failed: %v", err)
		}

		values, err := cs.Read(100, 2)
		if err != nil {
			t.Fatalf("coilStorage read failed: %v", err)
		}
		if len(values) != 2 {
			t.Fatalf("coilStorage read wrong length: got %d, want 2", len(values))
		}
		if values[0] != true || values[1] != false {
			t.Fatalf("coilStorage read failed")
		}
	})

	rs := &registerStorage{}
	t.Run("registerStorage", func(t *testing.T) {
		err := rs.Write(100, []uint16{1, 2})
		if err != nil {
			t.Fatalf("registerStorage write failed: %v", err)
		}

		values, err := rs.Read(100, 2)
		if err != nil {
			t.Fatalf("registerStorage read failed: %v", err)
		}
		if len(values) != 2 {
			t.Fatalf("registerStorage read wrong length: got %d, want 2", len(values))
		}
		if values[0] != 1 || values[1] != 2 {
			t.Fatalf("registerStorage read failed")
		}
	})
}
