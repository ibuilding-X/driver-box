package modbus

import (
	"errors"
)

const (
	FuncCodeReadCoils                  = 1
	FuncCodeReadDiscreteInputs         = 2
	FuncCodeReadHoldingRegisters       = 3
	FuncCodeReadInputRegisters         = 4
	FuncCodeWriteSingleCoil            = 5
	FuncCodeWriteSingleRegister        = 6
	FuncCodeWriteMultipleCoils         = 15
	FuncCodeWriteMultipleRegisters     = 16
	FuncCodeReadWriteMultipleRegisters = 23
	FuncCodeMaskWriteRegister          = 22
	FuncCodeReadFIFOQueue              = 24
)

const (
	ParityNone = "N"
	ParityOdd  = "O"
	ParityEven = "E" // default
)

const (
	BaudRate300    = 300
	BaudRate600    = 600
	BaudRate1200   = 1200
	BaudRate2400   = 2400
	BaudRate4800   = 4800
	BaudRate9600   = 9600 // default
	BaudRate14400  = 14400
	BaudRate19200  = 19200
	BaudRate38400  = 38400
	BaudRate56000  = 56000
	BaudRate57600  = 57600
	BaudRate115200 = 115200
	BaudRate128000 = 128000
	BaudRate153600 = 153600
	BaudRate230400 = 230400
	BaudRate256000 = 256000
	BaudRate460800 = 460800
	BaudRate921600 = 921600
)

const (
	DataBits5 = 5
	DataBits6 = 6
	DataBits7 = 7
	DataBits8 = 8 // default
)

const (
	StopBits1 = 1 // default
	StopBits2 = 2
)

const (
	rtuMinSize = 4
	rtuMaxSize = 256
)

var (
	ErrIllegalFunction    = errors.New("illegal function")
	ErrIllegalDataAddress = errors.New("illegal data address")
)

// FuncCodeText returns the text representation of a Modbus function code.
func FuncCodeText(code int) string {
	switch code {
	case FuncCodeReadCoils:
		return "Read Coils"
	case FuncCodeReadDiscreteInputs:
		return "Read Discrete Inputs"
	case FuncCodeReadHoldingRegisters:
		return "Read Holding Registers"
	case FuncCodeReadInputRegisters:
		return "Read Input Registers"
	case FuncCodeWriteSingleCoil:
		return "Write Single Coil"
	case FuncCodeWriteSingleRegister:
		return "Write Single Register"
	case FuncCodeWriteMultipleCoils:
		return "Write Multiple Coils"
	case FuncCodeWriteMultipleRegisters:
		return "Write Multiple Registers"
	case FuncCodeReadWriteMultipleRegisters:
		return "Read/Write Multiple Registers"
	case FuncCodeMaskWriteRegister:
		return "Mask Write Register"
	case FuncCodeReadFIFOQueue:
		return "Read FIFO Queue"
	default:
		return ""
	}
}
