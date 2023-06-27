package route

import (
	"driver-box/core/helper"
	"driver-box/driver/restful/controller"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"go.uber.org/zap"
	"net/http"
)

// Register 注册路由
func Register() {
	// 插件 REST API
	ps := controller.NewPluginStorage()
	addRoute("/plugin/cache/get", ps.Get(), http.MethodGet)
	addRoute("/plugin/cache/set", ps.Set(), http.MethodPost)

	// 核心配置 API
	conf := &controller.Config{}
	addRoute("/config/update", conf.Update(func() error {
		return nil
	}), http.MethodPost)

}

// addRoute 添加路由
func addRoute(route string, handler http.HandlerFunc, methods string) {
	err := service.RunningService().AddRoute(route, handler, methods)
	handelRouteErr(err)
}

// handelRouteErr 处理添加路由错误
func handelRouteErr(err error) {
	if err != nil {
		helper.Logger.Error("add route error", zap.Error(err))
	}
}
