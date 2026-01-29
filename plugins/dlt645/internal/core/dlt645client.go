package dlt645

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"go.uber.org/zap"
)

const (
	rtuExceptionSize = 5
)

// RTUClientProvider implements ClientProvider interface.
type Dlt645ClientProvider struct {
	serialPort
	logger
	*pool // 请求池,所有RTU客户端共用一个请求池
}

// check RTUClientProvider implements underlying method
var _ ClientProvider = (*Dlt645ClientProvider)(nil)

// 请求池,所有RTU客户端共用一个请求池
var rtuPool = newPool(rtuAduMaxSize)

// NewRTUClientProvider allocates and initializes a RTUClientProvider.
// it will use default /dev/ttyS0 19200 8 1 N and timeout 1000
func NewClientProvider() *Dlt645ClientProvider {
	p := &Dlt645ClientProvider{
		logger: newLogger("dlt645OutputLog =>"),
		pool:   rtuPool,
	}
	p.Timeout = SerialDefaultTimeout
	p.autoReconnect = SerialDefaultAutoReconnect
	return p
}

func (sf *protocolFrame) encodeRTUFrame(slaveID byte, pdu ProtocolDataUnit) ([]byte, error) {
	length := len(pdu.Data) + 4
	if length > rtuAduMaxSize {
		return nil, fmt.Errorf("dltcon: length of data '%v' must not be bigger than '%v'", length, rtuAduMaxSize)
	}
	requestAdu := sf.adu[:0:length]
	requestAdu = append(requestAdu, slaveID, pdu.FuncCode)
	requestAdu = append(requestAdu, pdu.Data...)
	checksum := crc16(requestAdu)
	requestAdu = append(requestAdu, byte(checksum), byte(checksum>>8))
	return requestAdu, nil
}

// decode extracts slaveID and PDU from RTU frame and verify CRC.
// 解码从RTU帧中提取slaveID和PDU，并验证CRC。
func decodeRTUFrame(adu []byte) (uint8, []byte, error) {
	if len(adu) < rtuAduMinSize { // Minimum size (including address, funcCode and CRC)
		return 0, nil, fmt.Errorf("dltcon: response length '%v' does not meet minimum '%v'", len(adu), rtuAduMinSize)
	}
	// Calculate checksum
	crc := crc16(adu[0 : len(adu)-2])
	expect := binary.LittleEndian.Uint16(adu[len(adu)-2:])
	if crc != expect {
		return 0, nil, fmt.Errorf("dltcon: response crc '%x' does not match expected '%x'", expect, crc)
	}
	// slaveID & PDU but pass crc
	return adu[0], adu[1 : len(adu)-2], nil
}

// Send request to the remote server,it implements on SendRawFrame
func (sf *Dlt645ClientProvider) Send(slaveID byte, request ProtocolDataUnit) (ProtocolDataUnit, error) {
	return ProtocolDataUnit{}, nil

}

// SendPdu send pdu request to the remote server
func (sf *Dlt645ClientProvider) SendPdu(slaveID byte, pduRequest []byte) ([]byte, error) {

	return nil, nil
}

// SendRawFrame send Adu frame
// SendRawFrame发送Adu帧。
func (dlt *Dlt645ClientProvider) SendRawFrame(request string) (response float64, err error) {
	request = strings.Replace(request, " ", "", -1)
	dlt.mu.Lock()
	defer dlt.mu.Unlock()

	// check  port is connected
	if !dlt.isConnected() {
		if dlt.autoReconnect == 0 {
			return 0, ErrClosedConnection
		}

		var err error
		for tryCnt := byte(0); tryCnt < dlt.autoReconnect; tryCnt++ {
			err = dlt.connect()
			if err == nil {
				break
			}
		}
		if err != nil {
			return 0, ErrConnectionFailed
		}
	}

	// Send the request
	//yoyo := []byte{104,01,00,68,35,82,00,104,17,04,51,51,52,51,108,22}
	serialMessage := HexStringToBytes(request)
	//serialMessage := []byte{00, 04, 00, 29, 00, 02, 224, 28}
	dlt.Debug("sending [% x]", serialMessage)

	//68 01 00 44 23 52 00 68 91 18 33 32 34 33 BC 36 33 33 33 33 33 33 33 33 33 33 BC 36 33 33 33 33 33 33 13 16
	//68 01 00 44 23 52 00 68 91 06 33 36 34 35 95 55 dd 16

	//test := fmt.Sprintf("wtf?sending [% x]", aduRequest)
	//fmt.Println(test)

	_, err = dlt.port.Write(serialMessage)
	if err != nil {
		_ = dlt.close()
		driverbox.Log().Error("dlt645 write port failed", zap.Error(err))
		return 0, err
	}

	var data [rtuAduMaxSize]byte

	//bytesToRead := calculateResponseLength(HexStringToBytes(request))
	//time.Sleep(dlt.calculateDelay(len(HexStringToBytes(request)) + bytesToRead))

	sum, _ := io.ReadFull(dlt.port, data[:])
	backData := fmt.Sprintf("[% x]", data[0:sum])

	return analysis(dlt, backData)
}

// 把字符串转换成字节数组
func HexStringToBytes(data string) []byte {
	if "" == data {
		return nil
	}
	data = strings.ToUpper(data)
	length := len(data) / 2
	dataChars := []byte(data)
	var byteData []byte = make([]byte, length)
	for i := 0; i < length; i++ {
		pos := i * 2
		byteData[i] = byte(charToByte(dataChars[pos])<<4 | charToByte(dataChars[pos+1]))
	}
	return byteData

}

func charToByte(c byte) byte {
	return (byte)(strings.Index("0123456789ABCDEF", string(c)))
}

// calculateDelay roughly calculates time needed for the next frame.
// See dltcon over Serial Line - Specification and Implementation Guide (page 13).
func (sf *Dlt645ClientProvider) calculateDelay(chars int) time.Duration {
	var characterDelay, frameDelay int // us

	if sf.BaudRate <= 0 || sf.BaudRate > 19200 {
		characterDelay = 750
		frameDelay = 1750
	} else {
		characterDelay = 15000000 / sf.BaudRate
		frameDelay = 35000000 / sf.BaudRate
	}
	return time.Duration(characterDelay*chars+frameDelay) * time.Microsecond
}

func calculateResponseLength(adu []byte) int {
	length := rtuAduMinSize
	switch adu[1] {
	case FuncCodeReadDiscreteInputs,
		FuncCodeReadCoils:
		count := int(binary.BigEndian.Uint16(adu[4:]))
		length += 1 + count/8
		if count%8 != 0 {
			length++
		}
	case FuncCodeReadInputRegisters,
		FuncCodeReadHoldingRegisters,
		FuncCodeReadWriteMultipleRegisters:
		count := int(binary.BigEndian.Uint16(adu[4:]))
		length += 1 + count*2
	case FuncCodeWriteSingleCoil,
		FuncCodeWriteMultipleCoils,
		FuncCodeWriteSingleRegister,
		FuncCodeWriteMultipleRegisters:
		length += 4
	case FuncCodeMaskWriteRegister:
		length += 6
	case FuncCodeReadFIFOQueue:
		// undetermined
	default:
	}
	return length
}
