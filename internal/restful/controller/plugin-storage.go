package controller

import (
	"encoding/json"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/helper/response"
	"io"
	"net/http"
)

type kv map[string]interface{}

type PluginStorage struct{}

func NewPluginStorage() *PluginStorage {
	return &PluginStorage{}
}

// Get 获取信息
// 返回数据结构：{"key":"value"}
func (ps *PluginStorage) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取查询 Key
		key := r.URL.Query().Get("key")
		if key == "" {
			response.String(w, http.StatusBadRequest, "key cannot be empty")
			return
		}

		// 响应
		value, ok := helper.PluginCacheMap.Load(key)
		if !ok {
			value = ""
		}
		obj := kv{key: value}
		response.JSON(w, http.StatusOK, obj)
		return
	}
}

// Set 存储信息
// body 示例：{"key", "value"}
func (ps *PluginStorage) Set() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 读取 body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			response.String(w, http.StatusInternalServerError, "read body error: %s", err)
			return
		}
		defer r.Body.Close()
		// 键值对解析
		var obj kv
		if err = json.Unmarshal(body, &body); err != nil {
			response.String(w, http.StatusBadRequest, "json decode error: %s", err)
			return
		}
		// 存储
		for key, value := range obj {
			helper.PluginCacheMap.Store(key, value)
		}
		// 响应
		w.WriteHeader(http.StatusOK)
		return
	}
}
