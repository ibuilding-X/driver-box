package modbus

import (
	"encoding/binary"
	"errors"
)

type Context struct {
	ProtocolMessage *ProtocolMessage
	serverHandler   ServerHandler

	Coils            *coilStorage
	DiscreteInputs   *coilStorage
	HoldingRegisters *registerStorage
	InputRegisters   *registerStorage
}

func (c *Context) GetBoolSlice() []bool {
	boolSlice := c.bytesToBoolSlice(c.ProtocolMessage.Values)
	if len(boolSlice) == 0 {
		return nil
	}

	if len(boolSlice) >= int(c.ProtocolMessage.Quantity) {
		return boolSlice[:c.ProtocolMessage.Quantity]
	}

	return boolSlice
}

func (c *Context) GetUint16Slice() []uint16 {
	uint16Slice := c.bytesToUint16Slice(c.ProtocolMessage.Values)
	if len(uint16Slice) == 0 {
		return nil
	}

	if len(uint16Slice) >= int(c.ProtocolMessage.Quantity) {
		return uint16Slice[:c.ProtocolMessage.Quantity]
	}

	return uint16Slice
}

func (c *Context) Response(values interface{}, err error) error {
	// error response
	if err != nil {
		return c.sendError(err)
	}

	// success response
	if values == nil {
		return c.send()
	}

	// handle []bool
	boolSlice, ok := values.([]bool)
	if ok {
		return c.sendBoolSlice(boolSlice)
	}

	// handle []uint16
	uint16Slice, ok := values.([]uint16)
	if ok {
		return c.sendUint16Slice(uint16Slice)
	}

	return errors.New("unsupported data type, only []bool and []uint16 are supported")
}

func (c *Context) send() error {
	return c.serverHandler.Send(c.ProtocolMessage)
}

func (c *Context) sendError(err error) error {
	switch {
	case errors.Is(err, ErrIllegalFunction):
		c.ProtocolMessage.ErrorCode = 1
	case errors.Is(err, ErrIllegalDataAddress):
		c.ProtocolMessage.ErrorCode = 2
	default:
		c.ProtocolMessage.ErrorCode = 4
	}

	return c.send()
}

func (c *Context) sendBoolSlice(data []bool) error {
	if len(data) == 0 {
		return errors.New("data is empty")
	}

	bs := c.boolSliceToBytes(data)
	c.ProtocolMessage.Values = bs
	c.ProtocolMessage.ValueLength = byte(len(bs))

	return c.send()
}

func (c *Context) sendUint16Slice(data []uint16) error {
	if len(data) == 0 {
		return errors.New("data is empty")
	}

	bs := c.uint16SliceToBytes(data)
	c.ProtocolMessage.Values = bs
	c.ProtocolMessage.ValueLength = byte(len(bs))

	return c.send()
}

func (c *Context) boolSliceToBytes(data []bool) []byte {
	if len(data) == 0 {
		return nil
	}

	length := len(data)
	byteLength := (length + 7) / 8
	result := make([]byte, byteLength)

	for i, value := range data {
		byteIndex := i / 8
		bitIndex := i % 8
		if value {
			result[byteIndex] |= 1 << bitIndex
		}
	}

	return result
}

func (c *Context) uint16SliceToBytes(data []uint16) []byte {
	if len(data) == 0 {
		return nil
	}

	result := make([]byte, 0, 2*len(data))
	for _, v := range data {
		result = binary.BigEndian.AppendUint16(result, v)
	}

	return result
}

func (c *Context) byteToBoolSlice(b byte) []bool {
	var bs [8]bool
	for i := 0; i < 8; i++ {
		if b>>i&1 == 1 {
			bs[i] = true
		}
	}
	return bs[:]
}

func (c *Context) bytesToBoolSlice(bs []byte) []bool {
	if len(bs) == 0 {
		return nil
	}

	var results []bool
	for _, b := range bs {
		results = append(results, c.byteToBoolSlice(b)...)
	}
	return results
}

func (c *Context) bytesToUint16Slice(bs []byte) []uint16 {
	if len(bs) == 0 {
		return nil
	}

	var results []uint16
	for i := 0; i < len(bs); i += 2 {
		if i+1 < len(bs) {
			results = append(results, binary.BigEndian.Uint16(bs[i:i+2]))
		} else {
			results = append(results, uint16(bs[i]))
		}
	}
	return results
}
