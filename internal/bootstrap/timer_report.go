package bootstrap

import (
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"time"
)

var interval = map[time.Duration]*time.Ticker{}

// 初始化定时上报任务
func initTimerReport() {

	c := time.Tick(3 * time.Second) //每个三秒检测一次,动态加载新的任务
	go func() {
		for {
			<-c
			for _, model := range helper.CoreCache.Models() {
				for _, point := range model.Points {
					timerReport := point.TimerReport
					if timerReport == "" {
						continue
					}
					duration, err := time.ParseDuration(timerReport)
					if err != nil {
						helper.Logger.Error("解析定时上报配置失败", zap.String("model", model.Name), zap.Error(err))
						continue
					}
					_, ok := interval[duration]
					if !ok {
						ticker := time.NewTicker(duration)
						interval[duration] = ticker
						go matchAndReportResource(duration, ticker)
					}
				}
			}
		}
	}()
}

/*
提取匹配的设备点位并触发上报
*/
func matchAndReportResource(duration time.Duration, ticker *time.Ticker) {
	for {
		<-interval[duration].C
		stop := true
		for _, model := range helper.CoreCache.Models() {
			//提取匹配当前定时周期的点位
			var points []config.PointBase
			for _, point := range model.Points {
				timerReport := point.TimerReport
				if timerReport == "" {
					continue
				}
				configDuration, _ := time.ParseDuration(timerReport)
				if configDuration == duration {
					points = append(points, point)
				}
			}
			if len(points) == 0 {
				continue
			}

			//遍历所有设备上报点位置
			stop = false
			for _, device := range model.Devices {
				var pd []plugin.PointData
				for _, point := range points {
					val, e := helper.DeviceShadow.GetDevicePoint(device.Name, point.Name)
					if e != nil {
						helper.Logger.Debug("获取设备上报点位失败", zap.String("device", device.Name), zap.String("point", point.Name), zap.Error(e))
						continue
					}
					pd = append(pd, plugin.PointData{
						PointName: point.Name,
						Value:     val,
					})
				}
				if len(pd) == 0 {
					continue
				}
				deviceData := plugin.DeviceData{
					DeviceName: device.Name,
					Values:     pd,
				}
				for _, export := range helper.Exports {
					export.ExportTo(deviceData)
				}
			}
		}
		//无有效任务，退出定时任务
		if stop {
			ticker.Stop()
			delete(interval, duration)
			break
		}
	}
}
