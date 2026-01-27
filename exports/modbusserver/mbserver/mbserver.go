package mbserver

import (
	"errors"

	"github.com/goburrow/serial"
	"github.com/ibuilding-x/driver-box/exports/modbusserver/mbserver/modbus"

	"sync"
)

type PropertyValue struct {
	Name  string
	Value interface{}
}

type HandlerFunc func(id string, propertyValues []PropertyValue)

type Server interface {
	Start() error                                                      // 启动服务监听
	Stop() error                                                       // 停止
	SetProperty(id string, property string, value interface{}) error   // 设置设备属性
	SetOnWriteHandler(func(id string, propertyValues []PropertyValue)) // 寄存器写入回调函数
}

type ServerConfig struct {
	URL           string `json:"url"` // 仅适用于 Modbus TCP, 示例：127.0.0.1:502【暂不支持 Modbus TCP 协议】
	serial.Config                     // 串口配置

	Models  []Model  `json:"models"`  // 模型列表（注意属性顺序）
	Devices []Device `json:"devices"` // 设备列表（注意设备顺序）
}

type serverImpl struct {
	config       *ServerConfig
	register     *Storage
	writeHandler func(id string, propertyValues []PropertyValue)
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

func (s *serverImpl) SetOnWriteHandler(f func(id string, propertyValues []PropertyValue)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writeHandler != nil {
		return
	}

	s.writeHandler = f

	// function code handlers
	s.slave.HandleFuncCode(modbus.FuncCodeReadHoldingRegisters, func(c *modbus.Context) {
		values, err := s.register.Read(c.ProtocolMessage.Address, c.ProtocolMessage.Quantity)
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

		// 写入寄存器
		if err := s.register.Write(address, values); err != nil {
			_ = c.Response(nil, err)
			return
		}

		// 地址解析，获取对应设备及属性
		groups := make(map[string]map[string]struct{}) // device => properties
		for i := uint16(0); i < quantity; i++ {
			id, property, err := s.register.RegisterInfo(address + i)
			if err != nil {
				continue
			}

			if _, ok := groups[id]; !ok {
				groups[id] = make(map[string]struct{})
			}

			groups[id][property] = struct{}{}
		}

		for id, properties := range groups {
			var pvs []PropertyValue
			for property, _ := range properties {
				// 获取属性值
				v, _ := s.register.GetProperty(id, property)
				pvs = append(pvs, PropertyValue{
					Name:  property,
					Value: v,
				})
			}

			// 执行回调
			go s.writeHandler(id, pvs)
		}
	})
}

func NewServer(conf *ServerConfig) Server {
	// Modbus RTU handler
	handler := modbus.NewRTUServerHandler(&conf.Config)

	// Register
	register := &Storage{}

	return &serverImpl{
		config:   conf,
		register: register,
		slave:    modbus.NewSlave(handler),
	}
}
