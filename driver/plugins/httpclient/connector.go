package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type connectorConfig struct {
	Url           string              `json:"url"`             // 请求地址
	Method        string              `json:"method"`          // 请求方法
	Headers       map[string][]string `json:"headers"`         // 请求头
	Form          map[string]string   `json:"form"`            // form-data
	DataUrlEncode map[string]string   `json:"data_url_encode"` // x-www-form-urlencoded
	DataRaw       string              `json:"data_raw"`        // raw
	Timeout       string              `json:"timeout"`         // 请求超时
}

type connector struct {
	plugin *Plugin
	config connectorConfig
	client *http.Client
}

// Release 释放资源
func (c *connector) Release() (err error) {
	return
}

// Send 发送请求
func (c *connector) Send(raw interface{}) (err error) {
	// 解析 lua 返回数据
	rawStr, ok := raw.(string)
	if !ok {
		return errors.New("lua script encode return data error")
	}
	var td transportationData
	if err = json.Unmarshal([]byte(rawStr), &td); err != nil {
		return
	}

	// 替换
	if td.Protocol.Form != nil {
		c.config.Form = td.Protocol.Form
	}
	if td.Protocol.DataUrlEncode != nil {
		c.config.DataUrlEncode = td.Protocol.DataUrlEncode
	}
	if td.Protocol.DataRaw != "" {
		c.config.DataRaw = td.Protocol.DataRaw
	}
	if td.Protocol.Headers != nil {
		c.config.Headers = td.Protocol.Headers
	}

	// 创建请求
	timeout, err := time.ParseDuration(c.config.Timeout)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	body, err := c.newRequestBody()
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(c.config.Method), c.config.Url, body)
	if err != nil {
		return
	}
	if c.config.Headers != nil {
		for k, v := range c.config.Headers {
			for _, value := range v {
				req.Header.Add(k, value)
			}
		}
	}

	// 发送请求
	res, err := c.client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	// 读取相应
	bodyByte, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	_, err = c.plugin.callback(c.plugin, string(bodyByte))
	return
}

func (c *connector) newRequestBody() (io.Reader, error) {
	if c.config.Form != nil {
		payload := &bytes.Buffer{}
		writer := multipart.NewWriter(payload)
		for k, v := range c.config.Form {
			_ = writer.WriteField(k, v)
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		return payload, nil
	}

	if c.config.DataUrlEncode != nil {
		vs := url.Values{}
		for k, v := range c.config.DataUrlEncode {
			vs.Add(k, v)
		}
		return strings.NewReader(vs.Encode()), nil
	}

	if c.config.DataRaw != "" {
		return strings.NewReader(c.config.DataRaw), nil
	}

	return nil, nil
}
