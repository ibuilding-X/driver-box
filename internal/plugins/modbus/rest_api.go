package modbus

import (
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	"net/http"
)

func InitRestAPI() {
	restful.HandleFunc("", func(request *http.Request) (any, error) {
		return nil, nil
	})

}
