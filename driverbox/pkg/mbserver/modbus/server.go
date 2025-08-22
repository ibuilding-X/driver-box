package modbus

import "io"

type ServerHandler interface {
	Listen() error
	Close() error
	HandleFunc(func(message *ProtocolMessage))
	Send(message *ProtocolMessage) error
}

type Transporter interface {
	io.ReadWriter
	Connect() error
	Close() error
}

type Packager interface {
	Encode(message ProtocolMessage) (bs []byte, err error)
	Decode(bs []byte) (message ProtocolMessage, err error)
}

type ProtocolMessage struct {
	TransactionIdentifier uint16 `json:"transactionIdentifier"` // tcp: transaction identifier
	ProtocolIdentifier    uint16 `json:"protocolIdentifier"`    // tcp: protocol identifier
	Length                uint16 `json:"length"`                // tcp: length of the following data
	UnitIdentifier        byte   `json:"unitIdentifier"`        // tcp: unit identifier
	SlaveId               byte   `json:"slaveId"`               // slave address
	FunctionCode          byte   `json:"functionCode"`          // function code
	Address               uint16 `json:"address"`               // start address
	Quantity              uint16 `json:"quantity"`              // quantity
	ValueLength           byte   `json:"valueLength"`           // value length
	Values                []byte `json:"values"`                // value bytes
	ErrorCode             byte   `json:"errorCode"`             // error code
	original              []byte // original ADU
}
