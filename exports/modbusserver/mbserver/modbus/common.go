package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// commonPackager
// [functionCode, CRCCheck) include function code, exclude CRC check
type commonPackager struct {
}

func (c *commonPackager) Encode(message ProtocolMessage) (bs []byte, err error) {
	// error code
	if message.ErrorCode != 0 {
		bs = append(bs, message.FunctionCode|0x80)
		bs = append(bs, message.ErrorCode)
		return
	}

	bs = append(bs, message.FunctionCode)

	switch int(message.FunctionCode) {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs, FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		bs = append(bs, message.ValueLength)
		bs = append(bs, message.Values...)
	case FuncCodeWriteSingleCoil, FuncCodeWriteSingleRegister:
		bs = binary.BigEndian.AppendUint16(bs, message.Address)
		bs = append(bs, message.Values...)
	case FuncCodeWriteMultipleCoils, FuncCodeWriteMultipleRegisters:
		bs = binary.BigEndian.AppendUint16(bs, message.Address)
		bs = binary.BigEndian.AppendUint16(bs, message.Quantity)
	default:
		err = fmt.Errorf("modbus: illegal function [%d]", message.FunctionCode)
	}

	return
}

func (c *commonPackager) Decode(bs []byte) (message ProtocolMessage, err error) {
	if len(bs) == 0 {
		return message, errors.New("modbus: illegal data")
	}

	// function code
	message.FunctionCode = bs[0]

	switch int(message.FunctionCode) {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs, FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		if len(bs) != 5 {
			err = errors.New("modbus: illegal data")
			return
		}
		message.Address = binary.BigEndian.Uint16(bs[1:3])
		message.Quantity = binary.BigEndian.Uint16(bs[3:5])
	case FuncCodeWriteSingleCoil, FuncCodeWriteSingleRegister: // write single register
		if len(bs) != 5 {
			err = errors.New("modbus: illegal data value")
			return
		}
		message.Address = binary.BigEndian.Uint16(bs[1:3])
		message.Quantity = 1
		message.Values = bs[3:5]
	case FuncCodeWriteMultipleCoils: // write multiple coils
		if len(bs) < 7 {
			err = errors.New("modbus: illegal data")
			return
		}
		message.Address = binary.BigEndian.Uint16(bs[1:3])
		message.Quantity = binary.BigEndian.Uint16(bs[3:5])
		message.ValueLength = bs[5]
		message.Values = bs[6:]
	case FuncCodeWriteMultipleRegisters: // write multiple registers
		if len(bs) < 8 {
			err = errors.New("modbus: illegal data")
			return
		}
		message.Address = binary.BigEndian.Uint16(bs[1:3])
		message.Quantity = binary.BigEndian.Uint16(bs[3:5])
		message.ValueLength = bs[5]
		message.Values = bs[6:]
	default:
		err = fmt.Errorf("modbus: illegal function [%d]", message.FunctionCode)
	}

	return
}
