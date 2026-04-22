package internal

import (
	"fmt"

	"github.com/ibuilding-x/driver-box/v2/driverbox"
	"github.com/robinson/gos7"
	"go.uber.org/zap"
)

type s7Client struct {
	config  *ConnectionConfig
	handler *gos7.TCPClientHandler
	client  gos7.Client
}

func newS7Client(config *ConnectionConfig) (*s7Client, error) {
	handler := gos7.NewTCPClientHandler(config.Address, config.Rack, config.Slot)
	if config.Timeout > 0 {
		handler.Timeout = config.Timeout
	}
	handler.IdleTimeout = config.Timeout * 2
	if err := handler.Connect(); err != nil {
		return nil, fmt.Errorf("connect s7 server error: %w", err)
	}
	client := gos7.NewClient(handler)
	return &s7Client{
		config:  config,
		handler: handler,
		client:  client,
	}, nil
}

func (sc *s7Client) ReadNodes(nodes []*NodeConfig) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if len(nodes) == 0 {
		return result, nil
	}
	for _, node := range nodes {
		value, err := sc.readNode(node)
		if err != nil {
			driverbox.Log().Warn("read node error", zap.String("point", node.PointName), zap.Error(err))
			continue
		}
		result[node.PointName] = value
	}
	return result, nil
}

func (sc *s7Client) readNode(node *NodeConfig) (interface{}, error) {
	switch node.DataType {
	case "BOOL":
		return sc.readBool(node)
	case "BYTE":
		return sc.readByte(node)
	case "WORD":
		return sc.readWord(node)
	case "DWORD":
		return sc.readDWord(node)
	case "INT":
		return sc.readInt(node)
	case "DINT":
		return sc.readDInt(node)
	case "REAL":
		return sc.readReal(node)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", node.DataType)
	}
}

func (sc *s7Client) readBool(node *NodeConfig) (bool, error) {
	size := 1
	buffer := make([]byte, size)
	var err error
	switch node.Area {
	case "DB":
		err = sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		err = sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		err = sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		err = sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return false, fmt.Errorf("unsupported area: %s", node.Area)
	}
	if err != nil {
		return false, err
	}
	bitIndex := node.Bit
	if bitIndex < 0 || bitIndex > 7 {
		bitIndex = 0
	}
	value := (buffer[0] & (1 << bitIndex)) != 0
	return value, nil
}

func (sc *s7Client) readByte(node *NodeConfig) (byte, error) {
	size := 1
	buffer := make([]byte, size)
	var err error
	switch node.Area {
	case "DB":
		err = sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		err = sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		err = sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		err = sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return 0, fmt.Errorf("unsupported area: %s", node.Area)
	}
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}

func (sc *s7Client) readWord(node *NodeConfig) (uint16, error) {
	size := 2
	buffer := make([]byte, size)
	var err error
	switch node.Area {
	case "DB":
		err = sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		err = sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		err = sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		err = sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return 0, fmt.Errorf("unsupported area: %s", node.Area)
	}
	if err != nil {
		return 0, err
	}
	var helper gos7.Helper
	var result uint16
	helper.GetValueAt(buffer, 0, &result)
	return result, nil
}

func (sc *s7Client) readDWord(node *NodeConfig) (uint32, error) {
	size := 4
	buffer := make([]byte, size)
	var err error
	switch node.Area {
	case "DB":
		err = sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		err = sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		err = sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		err = sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return 0, fmt.Errorf("unsupported area: %s", node.Area)
	}
	if err != nil {
		return 0, err
	}
	var helper gos7.Helper
	var result uint32
	helper.GetValueAt(buffer, 0, &result)
	return result, nil
}

func (sc *s7Client) readInt(node *NodeConfig) (int16, error) {
	size := 2
	buffer := make([]byte, size)
	var err error
	switch node.Area {
	case "DB":
		err = sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		err = sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		err = sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		err = sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return 0, fmt.Errorf("unsupported area: %s", node.Area)
	}
	if err != nil {
		return 0, err
	}
	var helper gos7.Helper
	var result int16
	helper.GetValueAt(buffer, 0, &result)
	return result, nil
}

func (sc *s7Client) readDInt(node *NodeConfig) (int32, error) {
	size := 4
	buffer := make([]byte, size)
	var err error
	switch node.Area {
	case "DB":
		err = sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		err = sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		err = sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		err = sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return 0, fmt.Errorf("unsupported area: %s", node.Area)
	}
	if err != nil {
		return 0, err
	}
	var helper gos7.Helper
	var result int32
	helper.GetValueAt(buffer, 0, &result)
	return result, nil
}

func (sc *s7Client) readReal(node *NodeConfig) (float32, error) {
	size := 4
	buffer := make([]byte, size)
	var err error
	switch node.Area {
	case "DB":
		err = sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		err = sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		err = sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		err = sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return 0, fmt.Errorf("unsupported area: %s", node.Area)
	}
	if err != nil {
		return 0, err
	}
	var helper gos7.Helper
	var result float32
	helper.GetValueAt(buffer, 0, &result)
	return result, nil
}

func (sc *s7Client) WriteNode(node *NodeConfig, value interface{}) error {
	switch node.DataType {
	case "BOOL":
		return sc.writeBool(node, value)
	case "BYTE":
		return sc.writeByte(node, value)
	case "WORD":
		return sc.writeWord(node, value)
	case "DWORD":
		return sc.writeDWord(node, value)
	case "INT":
		return sc.writeInt(node, value)
	case "DINT":
		return sc.writeDInt(node, value)
	case "REAL":
		return sc.writeReal(node, value)
	default:
		return fmt.Errorf("unsupported data type: %s", node.DataType)
	}
}

func (sc *s7Client) writeBool(node *NodeConfig, value interface{}) error {
	v, ok := value.(bool)
	if !ok {
		return fmt.Errorf("invalid value type for BOOL: %T", value)
	}
	size := 1
	buffer := make([]byte, size)
	err := sc.readBytes(node, buffer)
	if err != nil {
		return err
	}
	if v {
		buffer[0] |= (1 << node.Bit)
	} else {
		buffer[0] &= ^(1 << node.Bit)
	}
	return sc.writeBytes(node, buffer)
}

func (sc *s7Client) writeByte(node *NodeConfig, value interface{}) error {
	v, ok := value.(byte)
	if !ok {
		return fmt.Errorf("invalid value type for BYTE: %T", value)
	}
	buffer := []byte{v}
	return sc.writeBytes(node, buffer)
}

func (sc *s7Client) writeWord(node *NodeConfig, value interface{}) error {
	v, ok := value.(uint16)
	if !ok {
		return fmt.Errorf("invalid value type for WORD: %T", value)
	}
	size := 2
	buffer := make([]byte, size)
	var helper gos7.Helper
	helper.SetValueAt(buffer, 0, v)
	return sc.writeBytes(node, buffer)
}

func (sc *s7Client) writeDWord(node *NodeConfig, value interface{}) error {
	v, ok := value.(uint32)
	if !ok {
		return fmt.Errorf("invalid value type for DWORD: %T", value)
	}
	size := 4
	buffer := make([]byte, size)
	var helper gos7.Helper
	helper.SetValueAt(buffer, 0, v)
	return sc.writeBytes(node, buffer)
}

func (sc *s7Client) writeInt(node *NodeConfig, value interface{}) error {
	v, ok := value.(int16)
	if !ok {
		return fmt.Errorf("invalid value type for INT: %T", value)
	}
	size := 2
	buffer := make([]byte, size)
	var helper gos7.Helper
	helper.SetValueAt(buffer, 0, v)
	return sc.writeBytes(node, buffer)
}

func (sc *s7Client) writeDInt(node *NodeConfig, value interface{}) error {
	v, ok := value.(int32)
	if !ok {
		return fmt.Errorf("invalid value type for DINT: %T", value)
	}
	size := 4
	buffer := make([]byte, size)
	var helper gos7.Helper
	helper.SetValueAt(buffer, 0, v)
	return sc.writeBytes(node, buffer)
}

func (sc *s7Client) writeReal(node *NodeConfig, value interface{}) error {
	v, ok := value.(float32)
	if !ok {
		return fmt.Errorf("invalid value type for REAL: %T", value)
	}
	size := 4
	buffer := make([]byte, size)
	var helper gos7.Helper
	helper.SetValueAt(buffer, 0, v)
	return sc.writeBytes(node, buffer)
}

func (sc *s7Client) readBytes(node *NodeConfig, buffer []byte) error {
	size := len(buffer)
	switch node.Area {
	case "DB":
		return sc.client.AGReadDB(node.DB, node.Start, size, buffer)
	case "M":
		return sc.client.AGReadMB(node.Start, size, buffer)
	case "I":
		return sc.client.AGReadEB(node.Start, size, buffer)
	case "Q":
		return sc.client.AGReadAB(node.Start, size, buffer)
	default:
		return fmt.Errorf("unsupported area: %s", node.Area)
	}
}

func (sc *s7Client) writeBytes(node *NodeConfig, buffer []byte) error {
	size := len(buffer)
	switch node.Area {
	case "DB":
		return sc.client.AGWriteDB(node.DB, node.Start, size, buffer)
	case "M":
		return sc.client.AGWriteMB(node.Start, size, buffer)
	case "I":
		return sc.client.AGWriteEB(node.Start, size, buffer)
	case "Q":
		return sc.client.AGWriteAB(node.Start, size, buffer)
	default:
		return fmt.Errorf("unsupported area for write: %s", node.Area)
	}
}

func (sc *s7Client) Close() {
	if sc.handler != nil {
		sc.handler.Close()
	}
}
