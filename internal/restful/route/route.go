package route

import (
	controller2 "github.com/ibuilding-x/driver-box/internal/restful/controller"
	"net/http"
)

// Register 注册路由
func Register() error {
	// 插件 REST API
	ps := controller2.NewPluginStorage()

	http.HandleFunc("/plugin/cache/get", ps.Get())
	http.HandleFunc("/plugin/cache/set", ps.Set())
	// 核心配置 API
	conf := &controller2.Config{}
	http.HandleFunc("/config/update", conf.Update())

	return http.ListenAndServe(":8081", nil)
}
