package controller

import (
	"driver-box/core/helper/response"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type kv map[string]interface{}

type PluginStorage struct {
	m *sync.Map
}

func NewPluginStorage() *PluginStorage {
	return &PluginStorage{
		m: &sync.Map{},
	}
}

// Get 获取信息
// 返回数据结构：{"key":"value"}
func (ps *PluginStorage) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取查询 Key
		key := r.URL.Query().Get("key")
		var value string
		if key != "" {
			if v, ok := ps.m.Load(key); ok {
				value = fmt.Sprintf("%v", v)
			}
		}
		// 响应
		obj := kv{key: value}
		response.JSON(w, http.StatusOK, obj)
		return
	}
}

// Set 存储信息
func (ps *PluginStorage) Set() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取存储 key
		key := r.URL.Query().Get("key")
		if key == "" {
			response.String(w, http.StatusInternalServerError, "key cannot be empty")
			return
		}
		// 读取 body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			response.String(w, http.StatusInternalServerError, "read body error: %s", err)
			return
		}
		defer r.Body.Close()
		// 存储
		ps.m.Store(key, string(body))
		// 响应
		w.WriteHeader(http.StatusOK)
		return
	}
}
