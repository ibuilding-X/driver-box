package controller

import (
	"driver-box/core/helper"
	"driver-box/driver/common"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
)

type Config struct {
}

func (c *Config) Update(reload func() error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 接收核心配置
		body, err := io.ReadAll(r.Body)
		if err != nil {
			helper.Logger.Error("read request body error", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// 保存核心配置
		err = os.WriteFile(common.CoreConfigPath, body, 0644)
		if err != nil {
			helper.Logger.Error("save core config error", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 重载驱动插件
		if err = reload(); err != nil {
			helper.Logger.Error("reload core config error", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
