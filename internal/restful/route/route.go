package route

import (
	"github.com/ibuilding-x/driver-box/internal/restful/controller"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

const (
	V1Prefix string = "/api/v1/"
)

// Register 注册路由
func Register() error {
	router := httprouter.New()

	// 插件 REST API
	ps := controller.NewPluginStorage()
	router.GET(V1Prefix+"plugin/cache/get/:key", ps.Get)
	router.POST(V1Prefix+"plugin/cache/set", ps.Set)
	// 核心配置 API
	conf := &controller.Config{}
	router.POST(V1Prefix+"config/update", conf.Update)

	// 设备影子 API
	sdc := controller.NewShadow()
	router.GET(V1Prefix+"shadow/all", sdc.All)
	router.GET(V1Prefix+"shadow/device/:sn", sdc.QueryDevice)
	router.GET(V1Prefix+"shadow/device/:sn/:point", sdc.QueryDevicePoint)
	router.POST(V1Prefix+"shadow/device/:sn", sdc.UpdateDevicePoints)

	return http.ListenAndServe(":8081", nil)
}
