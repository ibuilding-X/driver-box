package route

import (
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	controller2 "github.com/ibuilding-x/driver-box/driverbox/restful/controller"
	"net/http"
)

const (
	V1Prefix string = "/api/v1/"
)

// Register 注册路由
func Register() error {
	// 插件 REST API
	ps := controller2.NewPluginStorage()
	restful.HandleFunc(V1Prefix+"plugin/cache/get", ps.Get)
	restful.HandleFunc(V1Prefix+"plugin/cache/set", ps.Set)
	// 核心配置 API
	conf := &controller2.Config{}
	restful.HandleFunc(V1Prefix+"config/update", conf.Update)

	// 设备影子 API
	sdc := &controller2.Shadow{}
	restful.HandleFunc(V1Prefix+"shadow/all", sdc.All)
	restful.HandleFunc(V1Prefix+"shadow/device", sdc.Device)
	restful.HandleFunc(V1Prefix+"shadow/devicePoint", sdc.DevicePoint)

	return http.ListenAndServe(":8081", nil)
}
