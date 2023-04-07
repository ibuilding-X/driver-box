package route

import (
	"driver-box/core/helper"
	"github.com/edgexfoundry/device-sdk-go/v2/pkg/service"
	"go.uber.org/zap"
	"net/http"
)

// Register 注册路由
func Register() {

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
