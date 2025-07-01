package mbslave

type Server interface {
	Start() error
	Stop() error
}

type Config struct {
}

type serverImpl struct {
	config  Config
	handler Handler
}

func (s *serverImpl) Start() error {
	//TODO implement me
	panic("implement me")
}

func (s *serverImpl) Stop() error {
	//TODO implement me
	panic("implement me")
}

func NewServer(config Config, handler Handler) Server {
	return &serverImpl{
		config:  config,
		handler: handler,
	}
}
