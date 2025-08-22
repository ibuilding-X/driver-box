package modbus

import (
	"sync"
)

type HandlerFunc func(c *Context)

type Slave struct {
	handler   ServerHandler
	functions map[int]HandlerFunc
	mu        sync.RWMutex

	coils            *coilStorage
	discreteInputs   *coilStorage
	holdingRegisters *registerStorage
	inputRegisters   *registerStorage
}

func (s *Slave) Listen() error {
	return s.handler.Listen()
}

func (s *Slave) Close() error {
	return s.handler.Close()
}

func (s *Slave) HandleFuncCode(code int, f HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handler.HandleFunc(s.handleProtocolMessageFunc)
	s.functions[code] = f
}

func (s *Slave) handleProtocolMessageFunc(message *ProtocolMessage) {
	funcCodeFunc, ok := s.functions[int(message.FunctionCode)]
	if !ok {
		message.ErrorCode = 1
		_ = s.handler.Send(message)
		return
	}

	ctx := s.createContext(message)
	funcCodeFunc(ctx)
}

func (s *Slave) createContext(message *ProtocolMessage) *Context {
	return &Context{
		ProtocolMessage:  message,
		serverHandler:    s.handler,
		Coils:            s.coils,
		DiscreteInputs:   s.discreteInputs,
		HoldingRegisters: s.holdingRegisters,
		InputRegisters:   s.inputRegisters,
	}
}

func NewSlave(handler ServerHandler) *Slave {
	return &Slave{
		handler:          handler,
		functions:        make(map[int]HandlerFunc),
		mu:               sync.RWMutex{},
		coils:            &coilStorage{},
		discreteInputs:   &coilStorage{},
		holdingRegisters: &registerStorage{},
		inputRegisters:   &registerStorage{},
	}
}
