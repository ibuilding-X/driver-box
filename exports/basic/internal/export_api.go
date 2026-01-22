package internal

import (
	"net/http"
	"strings"

	"github.com/ibuilding-x/driver-box/exports/basic/internal/restful"
	"github.com/ibuilding-x/driver-box/exports/basic/internal/restful/route"
)

func (export *Export) HandleFunc(method, pattern string, handler restful.Handler) {
	if strings.HasPrefix(pattern, "/") {
		restful.HandleFunc(method, pattern, handler)
	} else {
		restful.HandleFunc(method, route.V1Prefix+pattern, handler)
	}

}
func (export *Export) HandlerFunc(method, path string, handler http.HandlerFunc) {
	restful.HttpRouter.HandlerFunc(method, path, handler)
}
func (export *Export) HttpListen() string {
	return export.httpListen
}
