package mbserver

import (
	"errors"
	"fmt"
	"github.com/goburrow/serial"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/mbserver/modbus"
	"sync"
)

type HandlerFunc func(id string, property string, value interface{}) error

type Server interface {
	Start() error                                                          // 启动服务监听
	Stop() error                                                           // 停止
	SetProperty(id string, property string, value interface{}) error       // 设置设备属性
	SetOnWriteHandler(func(id string, property string, value interface{})) // 寄存器写入回调函数
}

type ServerConfig struct {
	URL           string `json:"url"` // 仅适用于 Modbus TCP, 示例：127.0.0.1:502【暂不支持 Modbus TCP 协议】
	serial.Config        // 串口配置

	Models  []Model  `json:"models"`  // 模型列表（注意属性顺序）
	Devices []Device `json:"devices"` // 设备列表（注意设备顺序）
}

type serverImpl struct {
	config       *ServerConfig
	register     Register
	writeHandler func(id string, property string, value interface{})
	slave        *modbus.Slave
	mu           sync.Mutex
}

func (s *serverImpl) Start() error {
	// init register
	if err := s.register.Initialize(s.config.Models, s.config.Devices); err != nil {
		return err
	}

	// modbus server
	return s.slave.Listen()
}

func (s *serverImpl) Stop() error {
	if err := s.slave.Close(); err != nil {
		return err
	}

	s.register = nil
	return nil
}

func (s *serverImpl) SetProperty(id string, property string, value interface{}) error {
	return s.register.SetProperty(id, property, value)
}

func (s *serverImpl) SetOnWriteHandler(f func(id string, property string, value interface{})) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writeHandler != nil {
		return
	}

	s.writeHandler = f

	// function code handlers
	s.slave.HandleFuncCode(modbus.FuncCodeReadHoldingRegisters, func(c *modbus.Context) {
		values, err := s.register.Read(c.ProtocolMessage.Address, c.ProtocolMessage.Quantity)
		fmt.Println(values, err)
		_ = c.Response(values, err)
	})
	s.slave.HandleFuncCode(modbus.FuncCodeWriteMultipleRegisters, func(c *modbus.Context) {
		address := c.ProtocolMessage.Address
		quantity := c.ProtocolMessage.Quantity
		values := c.GetUint16Slice()
		if len(values) != int(quantity) {
			_ = c.Response(nil, errors.New("invalid quantity"))
			return
		}

		for i := uint16(0); i < quantity; i++ {
			_ = s.register.Write(address, values[i])

			id, property, _, err := s.register.ParseAddress(address + i)
			if err != nil {
				continue
			}

			// get property value
			value, err := s.register.GetProperty(id, property)
			if err != nil {
				continue
			}

			s.writeHandler(id, property, value)
		}
	})
}

func NewServer(conf *ServerConfig) Server {
	// Modbus RTU handler
	handler := modbus.NewRTUServerHandler(&conf.Config)

	// Register
	register := NewRegister()

	return &serverImpl{
		config:   conf,
		register: register,
		slave:    modbus.NewSlave(handler),
	}
}
