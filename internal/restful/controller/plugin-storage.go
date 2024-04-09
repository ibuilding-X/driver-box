package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
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
func (ps *PluginStorage) Get(r *http.Request) (any, error) {
	// 获取查询 Key
	key := r.URL.Query().Get("key")
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}

	// 响应
	value, ok := helper.PluginCacheMap.Load(key)
	if !ok {
		value = ""
	}
	obj := kv{key: value}
	return obj, nil
}

// Set 存储信息
// body 示例：{"key", "value"}
func (ps *PluginStorage) Set(r *http.Request) (any, error) {
	// 读取 body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %s", err)
	}
	defer r.Body.Close()
	// 键值对解析
	var obj kv
	if err = json.Unmarshal(body, &body); err != nil {
		return nil, fmt.Errorf("json decode error: %s", err)
	}
	// 存储
	for key, value := range obj {
		helper.PluginCacheMap.Store(key, value)
	}
	// 响应
	return nil, nil
}
