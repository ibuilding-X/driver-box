package modbus

import (
	"errors"
	"github.com/goburrow/serial"
	"github.com/ibuilding-x/driver-box/driverbox/mbserver/modbus/crc"
	"sync"
)

type rtuTransporter struct {
	config *serial.Config
	port   serial.Port
	mu     sync.Mutex
}

func (r *rtuTransporter) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.port.Read(p)
}

func (r *rtuTransporter) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.port.Write(p)
}

func (r *rtuTransporter) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.port != nil {
		return nil
	}

	port, err := serial.Open(r.config)
	if err != nil {
		return err
	}

	r.port = port
	return nil
}

func (r *rtuTransporter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.port == nil {
		return nil
	}

	if err := r.port.Close(); err != nil {
		return err
	}

	r.port = nil
	return nil
}

type rtuPackager struct {
	commonPackager
}

func (r *rtuPackager) Encode(message ProtocolMessage) (bs []byte, err error) {
	bs, err = r.commonPackager.Encode(message)
	if err != nil {
		return
	}

	var result []byte
	// append slave id
	result = append(result, message.SlaveId)
	result = append(result, bs...)

	// append crc
	checksum := crc.Sum(result)
	result = append(result, byte(checksum), byte(checksum>>8))
	return result, err
}

func (r *rtuPackager) Decode(bs []byte) (message ProtocolMessage, err error) {
	if len(bs) < rtuMinSize || len(bs) > rtuMaxSize {
		return message, errors.New("modbus: the length of protocol message is between 4 and 256")
	}

	// crc
	if !r.CheckSum(bs) {
		return message, errors.New("modbus: the checksum does not match")
	}

	message, err = r.commonPackager.Decode(bs[1 : len(bs)-2])
	if err != nil {
		return
	}

	// slave id
	message.SlaveId = bs[0]

	// original
	message.original = bs
	return
}

func (r *rtuPackager) CheckSum(bs []byte) bool {
	length := len(bs)
	calc := crc.Sum(bs[0 : length-2])
	checksum := uint16(bs[length-1])<<8 | uint16(bs[length-2])
	if checksum != calc {
		return false
	}
	return true
}

type RTUServerHandler struct {
	rtuTransporter
	rtuPackager
	handler func(message *ProtocolMessage)
	mu      sync.RWMutex
}

func (r *RTUServerHandler) Listen() error {
	// open serial port
	if err := r.rtuTransporter.Connect(); err != nil {
		return err
	}

	// read
	var buff [rtuMaxSize]byte
	for {
		n, err := r.rtuTransporter.Read(buff[:])
		if err != nil {
			return err
		}

		if n == 0 {
			continue
		}

		message, err := r.rtuPackager.Decode(buff[:n])
		if err != nil {
			continue
		}

		r.mu.RLock()
		r.handler(&message)
		r.mu.RUnlock()
	}
}

func (r *RTUServerHandler) Close() error {
	return r.rtuTransporter.Close()
}

func (r *RTUServerHandler) HandleFunc(handler func(message *ProtocolMessage)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.handler != nil {
		return
	}

	r.handler = handler
}

func (r *RTUServerHandler) Send(message *ProtocolMessage) error {
	bs, err := r.rtuPackager.Encode(*message)
	if err != nil {
		return err
	}

	_, err = r.rtuTransporter.Write(bs)
	return err
}

func NewRTUServerHandler(config *serial.Config) *RTUServerHandler {
	handler := &RTUServerHandler{}
	handler.rtuTransporter.config = config
	return handler
}

func DefaultSerialConfig() serial.Config {
	return serial.Config{
		BaudRate: BaudRate9600,
		DataBits: DataBits8,
		StopBits: StopBits1,
		Parity:   ParityEven,
	}
}
