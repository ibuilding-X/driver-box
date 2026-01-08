package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ibuilding-x/driver-box/pkg/driverbox/common"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/helper/crontab"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/library"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/pkg/driverbox/plugin/callback"
	"go.uber.org/zap"
)

type connectorConfig struct {
	plugin.BaseConnection
	BaseUrl string        `json:"baseUrl"` // 基础URL
	Timeout int           `json:"timeout"` // 请求超时
	Timer   []timerConfig `json:"timer"`   //定时采集器
	Auth    string        `json:"auth"`    //认证信息
}

type timerConfig struct {
	Action   string `json:"action"`   // 定时采集器动作,lua方法名
	Duration string `json:"duration"` //采集周期
	duration time.Duration
	//上一次采集时间
	latestTime time.Time
}

type HttpRequest struct {
	Api    string            `json:"api"`
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type HttpResponse struct {
	HttpRequest
	StatusCode int `json:"statusCode"`
}

type TimerParam struct {
	Auth string `json:"auth"` //认证信息
}

type connector struct {
	plugin *Plugin
	config connectorConfig
	client *http.Client
}

// startServer 启动服务
func (c *connector) initCollectTask() (*crontab.Future, error) {
	if !c.config.Enable {
		helper.Logger.Warn("httpclient connector is not enable", zap.Any("connector", c.config))
		return nil, nil
	}
	if len(c.config.Timer) == 0 {
		helper.Logger.Warn("httpclient connector timer is empty", zap.Any("connector", c.config))
		return nil, nil
	}
	for i, timer := range c.config.Timer {
		duration, e := time.ParseDuration(timer.Duration)
		if e != nil {
			helper.Logger.Error("parse duration error", zap.Any("config", c.config))
			duration = time.Second * 5
		}
		c.config.Timer[i].duration = duration
	}
	actionParam := TimerParam{
		Auth: c.config.Auth,
	}
	bytes, e := json.Marshal(actionParam)
	if e != nil {
		helper.Logger.Error("json marshal error", zap.Any("config", c.config))
		return nil, e
	}
	action := string(bytes)
	return helper.Crontab.AddFunc("1s", func() {
		for i, timer := range c.config.Timer {
			//采集周期不满足，跳过本次
			if timer.latestTime.Add(timer.duration).After(time.Now()) {
				continue
			}
			c.config.Timer[i].latestTime = time.Now()
			payload, err := library.Protocol().Execute(c.config.ProtocolKey, timer.Action, action)
			if err != nil {
				helper.Logger.Error("execute protocol driver error", zap.Any("protocolKey", c.config.ProtocolKey), zap.Any("action", timer.Action), zap.Any("error", err))
				continue
			}
			var data HttpRequest
			json.Unmarshal([]byte(payload), &data)
			e := c.Send(data)
			if e != nil {
				helper.Logger.Error("send data error", zap.Any("data", data), zap.Any("error", e))
				continue
			}
		}
	})
}

// Release 释放资源
func (c *connector) Release() (err error) {
	return
}

// Send 发送请求
func (c *connector) Send(raw interface{}) (err error) {
	sendData := raw.(HttpRequest)

	// 创建请求
	//timeout := time.Duration(c.config.Timeout) * time.Millisecond
	//ctx, cancel := context.WithTimeout(context.Background(), timeout)
	//defer cancel()

	req, err := http.NewRequest(strings.ToUpper(sendData.Method), c.config.BaseUrl+sendData.Api, strings.NewReader(sendData.Body))
	if err != nil {
		return
	}
	if sendData.Header != nil {
		for k, v := range sendData.Header {
			req.Header.Add(k, v)
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
	responseHeader := make(map[string]string)
	for k, v := range res.Header {
		responseHeader[k] = v[0]
	}
	response := HttpResponse{
		HttpRequest: HttpRequest{
			Api:    sendData.Api,
			Method: sendData.Method,
			Header: responseHeader,
			Body:   string(bodyByte),
		},
		StatusCode: res.StatusCode,
	}
	deviceData, err := library.Protocol().Decode(c.config.ProtocolKey, response)
	if err != nil {
		return err
	}
	//自动添加设备
	common.WrapperDiscoverEvent(deviceData, c.config.ConnectionKey, ProtocolName)
	callback.ExportTo(deviceData)
	return nil
}

// Encode 编码数据
func (c *connector) Encode(deviceSn string, mode plugin.EncodeMode, values ...plugin.PointData) (res interface{}, err error) {
	return nil, common.NotSupportEncode
}

// Decode 解码数据，调用动态脚本解析
func (c *connector) Decode(raw interface{}) (res []plugin.DeviceData, err error) {
	return nil, common.NotSupportDecode
}
