package internal

import (
	"net/http"

	"github.com/ibuilding-x/driver-box/exports/basic/internal/restful"
)

type Api interface {
	HttpListen() string
	HandleFunc(method, pattern string, handler restful.Handler)
	HandlerFunc(method, path string, handler http.HandlerFunc)
}
