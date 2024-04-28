package route

import (
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	"github.com/ibuilding-x/driver-box/driverbox/restful/controller"
	"net/http"
)

const (
	V1Prefix string = "/api/v1/"
)

// Register 注册路由
func Register() error {
	// 插件 REST API
	ps := controller.NewPluginStorage()
	restful.HandleFunc(V1Prefix+"plugin/cache/get", ps.Get)
	restful.HandleFunc(V1Prefix+"plugin/cache/set", ps.Set)
	// 核心配置 API
	conf := &controller.Config{}
	restful.HandleFunc(V1Prefix+"config/update", conf.Update)

	// 设备影子 API
	sdc := &controller.Shadow{}
	restful.HandleFunc(V1Prefix+"shadow/all", sdc.All)
	restful.HandleFunc(V1Prefix+"shadow/device", sdc.Device)
	restful.HandleFunc(V1Prefix+"shadow/devicePoint", sdc.DevicePoint)

	//设备API
	d := &controller.Device{}
	restful.HandleFunc(DevicePointWrite, d.WritePoint)
	restful.HandleFunc(DevicePointsWrite, d.WritePoints)
	restful.HandleFunc(DevicePointRead, d.ReadPoint)

	return http.ListenAndServe(":8081", nil)
}
