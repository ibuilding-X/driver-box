package route

import (
	"driver-box/driver/restful/controller"
	"net/http"
)

// Register 注册路由
func Register() error {
	// 插件 REST API
	ps := controller.NewPluginStorage()

	http.HandleFunc("/plugin/cache/get", ps.Get())
	http.HandleFunc("/plugin/cache/set", ps.Set())
	// 核心配置 API
	conf := &controller.Config{}
	http.HandleFunc("/config/update", conf.Update())

	return http.ListenAndServe(":8081", nil)
}
