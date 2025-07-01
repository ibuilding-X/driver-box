package mbslave

import "errors"

// Modbus Error
var (
	ErrConfigurationError      = errors.New("configuration error")
	ErrRequestTimedOut         = errors.New("request timed out")
	ErrIllegalFunction         = errors.New("illegal function")
	ErrIllegalDataAddress      = errors.New("illegal data address")
	ErrIllegalDataValue        = errors.New("illegal data value")
	ErrServerDeviceFailure     = errors.New("server device failure")
	ErrAcknowledge             = errors.New("request acknowledged")
	ErrServerDeviceBusy        = errors.New("server device busy")
	ErrMemoryParityError       = errors.New("memory parity error")
	ErrGWPathUnavailable       = errors.New("gateway path unavailable")
	ErrGWTargetFailedToRespond = errors.New("gateway target device failed to respond")
	ErrBadCRC                  = errors.New("bad crc")
	ErrShortFrame              = errors.New("short frame")
	ErrProtocolError           = errors.New("protocol error")
	ErrBadUnitId               = errors.New("bad unit id")
	ErrBadTransactionId        = errors.New("bad transaction id")
	ErrUnknownProtocolId       = errors.New("unknown protocol identifier")
	ErrUnexpectedParameters    = errors.New("unexpected parameters")
)

var (
	ErrModelNotFound       = errors.New("model not found")
	ErrSlaveDeviceNotFound = errors.New("slave device not found")
)
