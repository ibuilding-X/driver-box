package logger

import (
	"sync"
)

var ChanWriter = &ChannelWriter{
	clients: &sync.Map{},
}

type ChannelWriter struct {
	clients *sync.Map
}

func (writer *ChannelWriter) Add(loggerChan chan []byte) {
	writer.clients.Store(loggerChan, true)
}
func (writer *ChannelWriter) Write(p []byte) (n int, err error) {
	writer.clients.Range(func(key, value interface{}) bool {
		loggerChan := key.(chan []byte)
		if len(loggerChan) < 100 {
			loggerChan <- p
		}
		return true
	})
	return 0, nil
}

func (writer ChannelWriter) Remove(loggerChan chan []byte) {
	writer.clients.Delete(loggerChan)
}
