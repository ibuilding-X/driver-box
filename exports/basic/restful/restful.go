package restful

import (
	"encoding/json"
	"net/http"

	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/exports/basic/restful/response"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var HttpRouter = httprouter.New()

// Handler 处理函数
type Handler func(*http.Request) (any, error)

// HandleFunc 注册处理函数
func HandleFunc(method, pattern string, handler Handler) {
	helper.Logger.Info("register api", zap.String("method", method), zap.String("pattern", pattern))
	HttpRouter.HandlerFunc(method, pattern, func(writer http.ResponseWriter, request *http.Request) {
		// 定义响应数据结构
		var data response.Common

		// 处理请求
		result, err := handler(request)
		if err != nil {
			// 定义错误信息
			data.ErrorMsg = err.Error()
			// 定义错误码
			if code, ok := errorCodes[err]; ok {
				data.ErrorCode = code
			} else {
				data.ErrorCode = errorCodes[UndefinedErr]
			}
		} else {
			data.Success = true
			data.ErrorCode = 200
			data.Data = result
		}

		// 设置响应头
		writer.Header().Set("Content-Type", "application/json")

		// 序列化响应数据
		b, err := json.Marshal(data)
		if err != nil {
			helper.Logger.Error("[api] json marshal fail", zap.Error(err))
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		// 写入响应数据
		_, err = writer.Write(b)
		if err != nil {
			helper.Logger.Error("[api] write response fail", zap.Error(err))
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
