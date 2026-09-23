package iec104

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/orglibs/go-iecp5/asdu"
)

func decodeWire(t *testing.T, raw []byte) *asdu.ASDU {
	t.Helper()
	a := asdu.NewEmptyASDU(asdu.ParamsWide)
	if err := a.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestDecodeSequenceAndQuality(t *testing.T) {
	// SQ=1: IOA 100/101/102 share a single address prefix.
	a := decodeWire(t, []byte{1, 0x83, 3, 0, 2, 0, 100, 0, 0, 1, 0x80, 0x41})
	samples, err := Decode(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 3 || samples[0].IOA != 100 || samples[1].IOA != 101 || samples[2].IOA != 102 || samples[0].CommonAddress != 2 || samples[0].Value != 1 {
		t.Fatalf("bad samples: %+v", samples)
	}
	if !samples[0].Valid() || samples[1].Valid() || samples[2].Valid() {
		t.Fatal("quality flags lost")
	}
	a.InfoObj = a.InfoObj[:2]
	if _, err := Decode(a); err == nil {
		t.Fatal("truncated ASDU accepted")
	}
}

func TestDecodeMonitoringFamilies(t *testing.T) {
	tests := []struct {
		typeID   byte
		payload  []byte
		expected any
	}{
		{3, []byte{2}, int(2)}, {5, []byte{0x7f, 0}, int(-1)}, {7, []byte{0x78, 0x56, 0x34, 0x12, 0}, uint32(0x12345678)},
		{9, []byte{0, 0x40, 0}, float64(.5)}, {11, []byte{0xfe, 0xff, 0}, int16(-2)}, {13, []byte{0, 0, 0xc0, 0x3f, 0}, float64(1.5)}, {15, []byte{42, 0, 0, 0, 0}, int32(42)}, {21, []byte{0, 0x80}, float64(-1)},
	}
	for _, tt := range tests {
		t.Run(asdu.TypeID(tt.typeID).String(), func(t *testing.T) {
			a := decodeWire(t, append([]byte{tt.typeID, 1, 3, 0, 1, 0, 1, 0, 0}, tt.payload...))
			samples, err := Decode(a)
			if err != nil {
				t.Fatal(err)
			}
			if len(samples) != 1 || samples[0].Value != tt.expected || !samples[0].Valid() {
				t.Fatalf("got %+v want %v", samples, tt.expected)
			}
		})
	}
}

func TestDecodeTimeAndNonFinite(t *testing.T) {
	stamp := time.Date(2026, 9, 20, 12, 13, 14, 150000000, time.UTC)
	raw := append([]byte{30, 1, 3, 0, 1, 0, 1, 0, 0, 1}, asdu.CP56Time2a(stamp, time.UTC, true)...)
	samples, err := Decode(decodeWire(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	if !samples[0].Time.Equal(stamp) || MonitoringFamily(samples[0].TypeID) != asdu.M_SP_NA_1 {
		t.Fatal(samples)
	}
	bits := make([]byte, 4)
	binary.LittleEndian.PutUint32(bits, math.Float32bits(float32(math.NaN())))
	raw = append([]byte{13, 1, 3, 0, 1, 0, 1, 0, 0}, bits...)
	raw = append(raw, 0)
	samples, err = Decode(decodeWire(t, raw))
	if err != nil || samples[0].Valid() {
		t.Fatal("NaN accepted")
	}
}

func TestCommandValidationAndEncoding(t *testing.T) {
	params := *asdu.ParamsWide
	params.OrigAddress = 7
	for _, tt := range []struct {
		command uint8
		value   any
		valid   bool
	}{
		{45, true, true}, {45, 2, false}, {46, 2, true}, {46, 0, false}, {48, .5, true}, {48, 1, false}, {49, -32768, true}, {49, 1.2, false}, {49, 32768, false}, {50, 1.25, true}, {50, math.NaN(), false}, {50, math.Inf(1), false}, {50, "bad", false},
	} {
		p := Point{Address: Address{2, 100}, Category: Signal, CommandType: tt.command}
		frame, err := Command(&params, p, tt.value, true)
		if (err == nil) != tt.valid {
			t.Fatalf("command %d value %v: %v", tt.command, tt.value, err)
		}
		if err == nil {
			raw, err := frame.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if raw[3] != 7 || raw[4] != 2 || raw[6] != 100 || raw[len(raw)-1]&0x80 == 0 {
				t.Fatalf("incorrect selected command %x", raw)
			}
		}
	}
}
