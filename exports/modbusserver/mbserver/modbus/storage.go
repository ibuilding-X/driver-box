package modbus

import (
	"fmt"
	"math"
	"sync"
)

type coilStorage struct {
	v  [65536]bool
	mu sync.RWMutex
}

func (c *coilStorage) Read(address uint16, quantity uint16) (values []bool, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err = verifyAddressQuantity(address, quantity); err != nil {
		return
	}

	values = make([]bool, quantity)
	copy(values, c.v[address:address+quantity])
	return
}

func (c *coilStorage) Write(address uint16, values []bool) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	quantity := uint16(len(values))
	if err = verifyAddressQuantity(address, quantity); err != nil {
		return
	}

	copy(c.v[address:address+quantity], values)
	return
}

type registerStorage struct {
	v  [65536]uint16
	mu sync.RWMutex
}

func (r *registerStorage) Read(address uint16, quantity uint16) (values []uint16, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err = verifyAddressQuantity(address, quantity); err != nil {
		return
	}

	values = make([]uint16, quantity)
	copy(values, r.v[address:address+quantity])
	return
}

func (r *registerStorage) Write(address uint16, values []uint16) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	quantity := uint16(len(values))
	if err = verifyAddressQuantity(address, quantity); err != nil {
		return
	}

	copy(r.v[address:address+quantity], values)
	return
}

func verifyAddressQuantity(address uint16, quantity uint16) error {
	if quantity < 1 || quantity > 2000 {
		return fmt.Errorf("modbus: quantity '%d' must be between '%d' and '%d'", quantity, 1, 2000)
	}

	if int(address+quantity) > math.MaxUint16+1 {
		return fmt.Errorf("modbus: quantity '%d' is out of range", quantity)
	}

	return nil
}
