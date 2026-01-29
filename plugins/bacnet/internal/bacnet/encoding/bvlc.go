package encoding

import (
	"github.com/ibuilding-x/driver-box/v2/plugins/bacnet/internal/bacnet/btypes"
)

// Bacnet Virtual Layer Control

func (e *Encoder) BVLC(b btypes.BVLC) error {
	// Set packet type
	e.write(b.Type)
	e.write(b.Function)
	e.write(b.Length)
	e.write(b.Data)
	return e.Error()
}

func (d *Decoder) BVLC(b *btypes.BVLC) error {
	d.decode(&b.Type)
	d.decode(&b.Function)
	d.decode(&b.Length)
	d.decode(&b.Data)
	return d.Error()
}
