package dlt645

import (
	"errors"
)

// ErrClosedConnection 连接已关闭
var ErrClosedConnection = errors.New("use of closed connection")
var ErrConnectionFailed = errors.New("create connection failed")
