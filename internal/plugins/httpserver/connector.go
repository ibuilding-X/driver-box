package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ibuilding-x/driver-box/driverbox/common"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type connectorConfig struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

type connector struct {
	scriptDir string      // 脚本目录名称
	ls        *lua.LState // lua 虚拟机
	plugin    *Plugin
	server    *http.Server
}

// Release 释放资源
func (c *connector) Release() (err error) {
	return c.server.Shutdown(context.Background())
}

// Send 被动接收数据模式，无需实现
func (c *connector) Send(raw interface{}) (err error) {
	return nil
}

// startServer 启动服务
func (c *connector) startServer(opts connectorConfig) {
	// 启动服务
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()
	app := gin.Default()
	// 通用路由
	app.NoRoute(func(ctx *gin.Context) {
		// 取 body
		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			c.plugin.logger.Error("http request read body error", zap.Error(err))
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    -1,
				"message": err.Error(),
			})
			return
		}
		// 重组协议数据
		data := protoData{
			Path:   ctx.Request.URL.Path,
			Method: ctx.Request.Method,
			Body:   string(body),
		}
		// 调用回调函数
		if err = callback.OnReceiveHandler(c, data.ToJSON()); err != nil {
			c.plugin.logger.Error("http_server callback error", zap.Error(err))
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"code":    -1,
				"message": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "ok",
		})
		return
	})

	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	c.server = &http.Server{
		Addr:    addr,
		Handler: app,
	}

	go func(addr string) {
		if err := c.server.ListenAndServe(); err != nil {
			c.plugin.logger.Error("start http server error", zap.Error(err))
		}
	}(addr)
}

// protoData 协议数据，框架重组交由动态脚本解析
type protoData struct {
	Path   string `json:"path"`   // 请求路径
	Method string `json:"method"` // 请求方法
	Body   string `json:"body"`   // 请求 body
	// todo 后续待扩充
}

// ToJSON 协议数据转 json 字符串
func (pd protoData) ToJSON() string {
	b, _ := json.Marshal(pd)
	return string(b)
}

// Encode 编码数据，无需实现
func (a *connector) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Decode 解码数据，调用动态脚本解析
func (a *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return helper.CallLuaConverter(a.ls, "decode", raw)
}
