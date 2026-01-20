package internal

import (
	bytes2 "bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ibuilding-x/driver-box/driverbox"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/exports/basic/restful"
	"github.com/ibuilding-x/driver-box/exports/basic/restful/route"
	"github.com/ibuilding-x/driver-box/pkg/event"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const (
	// ExecuteResultAllSuccess 全部成功
	ExecuteResultAllSuccess = "success"
	// ExecuteResultPartSuccess 部分成功
	ExecuteResultPartSuccess = "partSuccess"
	// ExecuteResultAllFail 全部失败
	ExecuteResultAllFail = "fail"
)

const (
	LinkConfigPath = "./config/linkedge" // 场景联动配置路径
)

// ErrActionListIsEmpty action 列表为空错误
var ErrActionListIsEmpty = errors.New("linkEdge action list cannot be empty")

type service struct {
	// 场景联动配置缓存
	configs map[string]Config
	// 定时任务
	schedules map[string]*cron.Cron
	//点位触发器
	triggerConditions map[string][]DevicePointCondition

	envConfig EnvConfig
}

func (s *service) NewService() error {
	// 创建联动场景
	restful.HandleFunc(http.MethodPost, route.LinkEdgeCreate, func(request *http.Request) (any, error) {
		data, err := readBody(request)
		if err != nil {
			err = fmt.Errorf("incoming reading ignored. Unable to read request body: %s", err.Error())
			helper.Logger.Error(err.Error())
			return false, err
		}
		err = s.Create(data)
		if err != nil {
			err = fmt.Errorf("create linkEdge error: %s", err.Error())
			helper.Logger.Error(err.Error())
			return false, err
		}
		return true, nil
	})

	// 预览联动场景
	restful.HandleFunc(http.MethodPost, route.LinkEdgeTryTrigger, func(request *http.Request) (any, error) {
		// 读取请求参数
		data, err := readBody(request)
		if err != nil {
			return false, err
		}
		// 解析数据
		var config Config
		err = json.Unmarshal(data, &config)
		if err != nil {
			return false, err
		}
		// 执行
		err = s.triggerLinkEdge("", 0, config)
		if err != nil {
			err = fmt.Errorf("preview linkEdge error: %s", err.Error())
			helper.Logger.Error(err.Error())
			return false, err
		}
		return true, nil
	})

	//删除联动场景
	restful.HandleFunc(http.MethodPost, route.LinkEdgeDelete, func(request *http.Request) (any, error) {
		err := s.Delete(request.FormValue("id"))
		return err != nil, err
	})

	//触发联动场景
	restful.HandleFunc(http.MethodPost, route.LinkEdgeTrigger, func(request *http.Request) (any, error) {
		helper.Logger.Info(fmt.Sprintf("trigger linkEdge:%s from: %s", request.FormValue("id"), request.FormValue("source")))
		err := s.TriggerLinkEdge(request.FormValue("id"))
		return err != nil, err
	})

	//查看场景联动
	restful.HandleFunc(http.MethodPost, route.LinkEdgeGet, func(request *http.Request) (any, error) {
		helper.Logger.Info(fmt.Sprintf("get linkEdge:%s", request.FormValue("id")))
		return s.getLinkEdge(request.FormValue("id"))
	})

	// 查看场景联动列表
	restful.HandleFunc(http.MethodGet, route.LinkEdgeList, func(request *http.Request) (any, error) {
		// 获取查询参数
		tag := request.URL.Query().Get("tag")
		return s.GetList(tag)
	})

	restful.HandleFunc(http.MethodPost, route.LinkEdgeUpdate, func(request *http.Request) (any, error) {
		body, err := readBody(request)
		if err != nil {
			return false, err
		}
		var model Config
		err = json.Unmarshal(body, &model)
		if err != nil {
			return false, err
		}
		_, err = s.getLinkEdge(model.ID)
		if err != nil {
			return false, err
		}
		err = s.Update(body)
		if err != nil {
			return false, err
		} else {
			return true, nil
		}
	})

	//更新场景联动状态
	restful.HandleFunc(http.MethodPost, route.LinkEdgeStatus, func(request *http.Request) (any, error) {
		helper.Logger.Info(fmt.Sprintf("get linkEdge:%s", request.FormValue("id")))
		config, err := s.getLinkEdge(request.FormValue("id"))
		if err != nil {
			return false, fmt.Errorf("unable to find link edge: %s", err)
		}
		enable := request.FormValue("enable")
		if enable != "true" && enable != "false" {
			return false, fmt.Errorf("invalid formField[enable] value")
		}
		config.Enable = "true" == enable

		bf := bytes2.NewBuffer([]byte{})
		jsonEncoder := json.NewEncoder(bf)
		jsonEncoder.SetEscapeHTML(false)
		err = jsonEncoder.Encode(config)
		if err != nil {
			return false, fmt.Errorf("encode %v error: %v", config, err)
		}
		if err = s.Update(bf.Bytes()); err == nil {
			return true, nil
		} else {
			return false, fmt.Errorf("update %v error: %v", config, err)
		}
	})

	// 获取最后一次执行的场景信息
	restful.HandleFunc(http.MethodGet, route.LinkEdgeGetLast, func(r *http.Request) (any, error) {
		return s.GetLast()
	})

	//启动场景联动
	configs, e := s.GetList()
	if e != nil {
		return e
	}
	for _, config := range configs {
		e = s.registerTrigger(config.ID)
		if e != nil {
			return e
		}
	}

	return nil
}

// Create 创建场景联动规则
func (s *service) Create(bytes []byte) error {
	var model Config
	e := json.Unmarshal(bytes, &model)
	if e != nil {
		return e
	}
	if _, exists := s.configs[model.ID]; exists {
		return errors.New("linkEdge id is exists")
	}

	// 空 Action 校验
	if len(model.Action) == 0 {
		return ErrActionListIsEmpty
	}

	//持久化
	// fix: 偶现写入的文件内容为空
	file := path.Join(s.envConfig.ConfigPath, model.ID+".json")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(bytes); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}

	//启动场景联动
	e = s.registerTrigger(model.ID)
	if e != nil {
		//注册触发器存在异常,清理脏数据
		_ = s.Delete(model.ID)
		return e
	}
	return nil
}

// 注册触发器
func (s *service) registerTrigger(id string) error {
	model, e := s.getLinkEdge(id)
	if e != nil {
		return e
	}
	//注册触发器
	for _, trigger := range model.Trigger {
		switch trigger.Type {
		case TriggerTypeSchedule:
			schedule, exists := s.schedules[model.ID]
			if !exists {
				schedule = cron.New()
				s.schedules[model.ID] = schedule
				schedule.Start()
			}
			_, e = schedule.AddFunc(trigger.Cron, func() {
				s.TriggerLinkEdge(model.ID)
			})
			if e != nil {
				return e
			} else {
				helper.Logger.Info(fmt.Sprintf("add schedule trigger:%v", trigger.Cron))
			}
			break
		case TriggerTypeDevicePoint:
			//注册eKuiper监听设备点位状态
			if len(trigger.DeviceID) == 0 || len(trigger.DevicePoint) == 0 || len(trigger.Condition) == 0 || len(trigger.Value) == 0 {
				bs, _ := json.Marshal(trigger.DevicePointTrigger)
				return errors.New("invalid trigger:" + string(bs))
			}

			//定义触发器
			triggers, _ := s.triggerConditions[id]
			triggers = append(triggers, trigger.DevicePointCondition)
			s.triggerConditions[id] = triggers
			break
		case TriggerTypeDeviceEvent:
			break
		default:
			bs, _ := json.Marshal(trigger)
			helper.Logger.Error(fmt.Sprintf("unsupport trigger type:%s", string(bs)))
		}
	}
	return nil
}

// Delete 删除场景联动规则
func (s *service) Delete(id string) error {
	if len(id) == 0 {
		return errors.New("id is nil")
	}
	helper.Logger.Info(fmt.Sprintf("delete linkEdge:%v", id))

	//删除配置
	delete(s.configs, id)
	delete(s.triggerConditions, id)
	file := path.Join(s.envConfig.ConfigPath, id+".json")
	e := os.Remove(file)
	if e != nil {
		return e
	}

	// 清理当前场景ID的所有时间表触发器
	helper.Logger.Info(fmt.Sprintf("clear schedule trigger..."))
	task, exists := s.schedules[id]
	if exists {
		task.Stop()
		delete(s.schedules, id)
	}
	return nil
}

// Update UpdateLinkEdgeStatus 调整联动规则状态,用于启停控制
func (s *service) Update(bytes []byte) error {
	var model Config
	e := json.Unmarshal(bytes, &model)
	if e != nil {
		return e
	}
	// action 为空校验
	if len(model.Action) <= 0 {
		return ErrActionListIsEmpty
	}
	e = s.Delete(model.ID)
	if e != nil {
		return e
	}
	return s.Create(bytes)
}

// TriggerLinkEdge 触发场景联动规则
// id: 场景联动ID
// source: 场景触发来源
func (s *service) TriggerLinkEdge(id string) error {
	//记录场景执行记录
	e := s.triggerLinkEdge(id, 0)
	if e != nil {
		helper.Logger.Error(fmt.Sprintf("linkEdge:%s trigger", e.Error()))
		return e
	}
	//缓存场景联动执行时间
	config, e := s.getLinkEdge(id)
	if e == nil {
		config.ExecuteTime = time.Now()
		s.configs[id] = config
	}
	return e
}

// depth:联动深度
func (s *service) triggerLinkEdge(id string, depth int, conf ...Config) error {
	if depth > 10 {
		return errors.New("execute level is too deep, max deep:" + strconv.Itoa(depth))
	}
	var config Config
	var e error
	if len(conf) > 0 {
		config = conf[0]
	} else {
		// 核对触发器
		config, e = s.getLinkEdge(id)
		if e != nil {
			driverbox.TriggerEvents(event.UnknownLinkEdge, "id", id)
			return errors.New("get linkEdge error: " + e.Error())
		}
		//helper.Logger.Info(fmt.Sprintf("linkEdge:%v", config))
	}

	if !config.Enable {
		return errors.New("linkEdge is disable now")
	}
	//静默期判断
	if config.SilentPeriod > 0 {
		//缓存场景联动执行时间
		if time.Now().Add(-time.Duration(config.SilentPeriod) * time.Second).Before(config.ExecuteTime) {
			return errors.New("execute frequency is too high")
		}
	}
	//校验执行条件
	e = s.checkConditions(config.Condition)
	if e != nil {
		return errors.New("check condition error: " + e.Error())
	}

	//组合相同设备的点位action
	actions := make(map[string][]plugin.PointData)
	sucCount := 0
	//执行动作
	for _, action := range config.Action {
		//判断执行动作是否匹配条件
		e = s.checkConditions(action.Condition)
		if e != nil {
			sucCount++
			continue
		}

		switch action.Type {
		// 设置设备点位
		case ActionTypeDevicePoint:
			deviceID := action.DeviceID
			if _, ok := actions[deviceID]; !ok {
				actions[deviceID] = make([]plugin.PointData, 0)
			}

			// （单点位设置）兼容旧版本
			if action.DevicePoint != "" && action.Value != "" {
				actions[deviceID] = append(actions[deviceID], plugin.PointData{
					PointName: action.DevicePoint,
					Value:     action.Value,
				})
			}

			// 多点位设置
			if len(action.Points) != 0 {
				for _, point := range action.Points {
					actions[deviceID] = append(actions[deviceID], plugin.PointData{
						PointName: point.Point,
						Value:     point.Value,
					})
				}
			}
		case ActionTypeLinkEdge:
			sucCount++
			go s.triggerLinkEdge(action.ID, depth+1)
		default:
			bytes, _ := json.Marshal(action)
			helper.Logger.Error(fmt.Sprintf("unsupport action:%s", string(bytes)))
		}

		//场景执行后休眠指定时长
		if len(action.Sleep) > 0 {
			d, err := time.ParseDuration(action.Sleep)
			if err == nil {
				time.Sleep(d)
			}
		}
	}
	//遍历执行actions,按连接分组
	connectGroup := make(map[string]map[string][]plugin.PointData)
	for deviceId, points := range actions {
		// 跳过未知设备
		if !driverbox.Shadow().HasDevice(deviceId) {
			// 事件信息：场景ID、设备ID
			driverbox.TriggerEvents(event.UnknownDevice, id, deviceId)
			continue
		}
		device, ok := driverbox.CoreCache().GetDevice(deviceId)
		if !ok {
			helper.Logger.Error("get device error", zap.String("deviceId", deviceId))
			continue
		}
		group, ok := connectGroup[device.ConnectionKey]
		if !ok {
			group = make(map[string][]plugin.PointData)
			connectGroup[device.ConnectionKey] = group
		}
		group[deviceId] = points
	}
	var wg sync.WaitGroup
	for _, devices := range connectGroup {
		wg.Add(1)
		go func(ds map[string][]plugin.PointData) {
			defer wg.Done()
			for deviceId, points := range ds {
				// 跳过离线设备，避免阻塞场景执行
				device, ok := driverbox.Shadow().GetDevice(deviceId)
				if !ok || !device.Online {
					helper.Logger.Error("device offline, skip action", zap.String("deviceId", deviceId), zap.String("linkedge", id))
					continue
				}
				err := driverbox.WritePoints(deviceId, points)
				if err != nil {
					helper.Logger.Error("execute linkEdge error", zap.String("linkEdge", id),
						zap.String("deviceId", deviceId), zap.Any("points", points), zap.Error(err))
				} else {
					sucCount = sucCount + len(points)
					helper.Logger.Info(fmt.Sprintf("execute linkEdge:%s action", id))
				}
			}
		}(devices)
	}
	wg.Wait()
	//预览情况下未持久化场景联动，id为空
	if id != "" {
		// value:全部成功\部分成功\全部失败
		if sucCount == len(config.Action) {
			driverbox.TriggerEvents(event.EventCodeLinkEdgeTrigger, id, ExecuteResultAllSuccess)
		} else if sucCount == 0 {
			driverbox.TriggerEvents(event.EventCodeLinkEdgeTrigger, id, ExecuteResultAllFail)
		} else {
			driverbox.TriggerEvents(event.EventCodeLinkEdgeTrigger, id, ExecuteResultPartSuccess)
		}
	}

	return nil
}

func (s *service) checkConditions(conditions []Condition) error {
	//优先执行点位持续时间条件校验
	err := s.checkListTimeCondition(conditions)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, condition := range conditions {
		helper.Logger.Info(fmt.Sprintf("check condition:%v", condition))
		if condition.Type == ConditionTypeLastTime {
			continue
		}
		switch condition.Type {
		case ConditionTypeDevicePoint:
			//注册eKuiper监听设备点位状态
			if len(condition.DeviceID) == 0 || len(condition.DevicePoint) == 0 || len(condition.Condition) == 0 || len(condition.Value) == 0 {
				bytes, _ := json.Marshal(condition.DevicePointCondition)
				return errors.New("invalid trigger:" + string(bytes))
			}
			pointValue, err := driverbox.Shadow().GetDevicePoint(condition.DeviceID, condition.DevicePoint)
			if err != nil {
				return fmt.Errorf("get device:%v point:%v value error:%v", condition.DeviceID, condition.DevicePoint, err)
			}
			//helper.Logger.Info(fmt.Sprintf("point value:%s", point))
			err = s.checkConditionValue(condition.DevicePointCondition, pointValue)
			if err != nil {
				return err
			}
		case ConditionTypeExecuteTime:
			if condition.Begin > now {
				return errors.New("execution time has not started")
			}
			if condition.End < now {
				return errors.New("execution time has expired")
			}
		case ConditionTypeDateInterval:
			if condition.BeginDate == "" || condition.EndDate == "" {
				return nil
			}

			begin, err := s.parseDate(condition.BeginDate)
			if err != nil {
				return errors.New("execution begin date parse error")
			}
			end, err := s.parseDate(condition.EndDate)
			if err != nil {
				return errors.New("execution end date parse error")
			}

			yearDay := time.Now().YearDay()
			if end.After(begin) {
				if yearDay >= begin.YearDay() && yearDay <= end.YearDay() {
					return nil
				}
			} else {
				if (yearDay >= 1 && yearDay <= end.YearDay()) || (yearDay >= begin.YearDay() && yearDay <= 366) {
					return nil
				}
			}

			return errors.New("execution time is not yet available")
		case ConditionTypeYears:
			if !condition.YearsCondition.Verify(time.Now().Year()) {
				return errors.New("mismatch years condition")
			}
		case ConditionTypeMonths:
			if !condition.MonthsCondition.Verify(int(time.Now().Month())) {
				return errors.New("mismatch months condition")
			}
		case ConditionTypeDays:
			if !condition.DaysCondition.Verify(time.Now().Day()) {
				return errors.New("mismatch days condition")
			}
		case ConditionTypeWeeks:
			if !condition.WeeksCondition.Verify(int(time.Now().Weekday())) {
				return errors.New("mismatch weeks condition")
			}
		case ConditionTypeTimes:
			if !condition.TimesCondition.Verify(time.Now()) {
				return errors.New("mismatch times condition")
			}
		}
	}
	return nil
}

func (s *service) checkListTimeCondition(conditions []Condition) error {
	//return errors.New("功能未迁移...")
	return nil
}

func (s *service) checkConditionValue(condition DevicePointCondition, pointValue interface{}) error {
	helper.Logger.Info(fmt.Sprintf("checkConditionValue condition:%v, pointValue:%v", condition, pointValue))
	e := errors.New(fmt.Sprintf("condition check fail. expect %v%v%v ,actual value=%v", condition.DevicePoint, condition.Condition, condition.Value, pointValue))
	switch pointValue.(type) {
	case string:
		switch condition.Condition {
		case ConditionEq:
			if condition.Value != pointValue {
				return e
			}
			break
		case ConditionNe:
			if condition.Value == pointValue {
				return e
			}
		default:
			return fmt.Errorf("unSupport condition type:%v for string point:%v ,value:%v", condition.Condition, condition.DevicePoint, pointValue)
		}
		return nil
	default:
		pointValue, e1 := strconv.ParseFloat(fmt.Sprintf("%v", pointValue), 32)
		if e1 != nil {
			return e1
		}
		//数值类型比较
		conditionValue, e1 := strconv.ParseFloat(condition.Value, 32)
		if e1 != nil {
			return e1
		}

		switch condition.Condition {
		case ConditionEq:
			if conditionValue != pointValue {
				return e
			}
			break
		case ConditionNe:
			if conditionValue == pointValue {
				return e
			}
			break
		case ConditionGt:
			if conditionValue >= pointValue {
				return e
			}
			break
		case ConditionGe:
			if conditionValue > pointValue {
				return e
			}
			break
		case ConditionLt:
			if conditionValue <= pointValue {
				return e
			}
			break
		case ConditionLe:
			if conditionValue < pointValue {
				return e
			}
			break
		}
	}

	return nil
}

func (s *service) getLinkEdge(id string) (Config, error) {
	config, exists := s.configs[id]
	if exists {
		return config, nil
	}

	config = Config{}
	fileName := filepath.Join(s.envConfig.ConfigPath, id+".json")
	f, err := os.Open(fileName)
	if err != nil {
		return config, err
	}
	defer f.Close()

	body, err := io.ReadAll(f)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(body, &config)
	s.configs[id] = config
	return config, err
}

func (s *service) GetList(tag ...string) ([]Config, error) {
	files := make([]string, 0)
	//若目录不存在，则自动创建
	_, err := os.Stat(s.envConfig.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(s.envConfig.ConfigPath, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	filepath.Walk(s.envConfig.ConfigPath, func(path string, d fs.FileInfo, err error) error {
		if !d.IsDir() {
			files = append(files, d.Name())
		}
		return nil
	})

	var configs []Config

	for _, key := range files {
		id := strings.TrimSuffix(key, ".json")
		config, err := s.getLinkEdge(id)
		if err != nil {
			return configs, err
		} else {
			if len(tag) > 0 && tag[0] != "" {
				if config.ExistTag(tag[0]) {
					configs = append(configs, config)
				}
				continue
			}
			configs = append(configs, config)
		}

	}
	return configs, nil
}

// GetLast 获取最后一次执行的场景信息
func (s *service) GetLast() (c Config, err error) {
	defaultTime := time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
	configs, err := s.GetList()
	if err != nil {
		return
	}
	for _, config := range configs {
		if config.ExecuteTime.After(defaultTime) || config.ExecuteTime.Equal(defaultTime) {
			defaultTime = config.ExecuteTime
			c = config
		}
	}
	// 判断执行时间，若执行时间为空，则返回空
	if c.ExecuteTime.IsZero() {
		return Config{}, nil
	}
	return
}

func readBody(request *http.Request) ([]byte, error) {
	defer request.Body.Close()
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("no request body provided")
	}

	return body, nil
}

// parseDate 解析日期
// 修复：02-29 问题
func (s *service) parseDate(d string) (time.Time, error) {
	year := time.Now().Year()
	if d == "02-29" && (year%4 != 0) {
		d = "02-28"
	}
	return time.Parse("2006-01-02", fmt.Sprintf("%d-%s", year, d))
}

// Preview 预览场景
// 提示：不真实创建场景，仅看执行效果使用
func (s *service) Preview(config Config) error {
	// 记录场景执行记录
	return s.triggerLinkEdge("", 0, config)
}

func (s *service) devicePointTriggerHandler(deviceData plugin.DeviceData, handleDuration bool) {
	// 循环点位
	for _, pointData := range deviceData.Values {
		// 循环场景
		for sceneID, conditions := range s.triggerConditions {
			helper.Logger.Debug("check linkedge condition ", zap.String("id", sceneID))
			// 循环条件
			for i, condition := range conditions {
				if condition.DeviceID != deviceData.ID || condition.DevicePoint != pointData.PointName {
					continue
				}

				switch handleDuration {
				case true: // 处理持续时间条件
					// 仅处理持续条件的触发条件（即 DevicePointCondition.Duration > 0）
					if condition.Duration <= 0 {
						continue
					}

					// 条件验证
					checkResult := s.checkConditionValue(condition, pointData.Value)
					if checkResult != nil { // 不满足条件
						s.triggerConditions[sceneID][i].DurationStartTime = time.Time{}
						continue
					}

					// 满足条件
					startTime := s.triggerConditions[sceneID][i].DurationStartTime
					if startTime.IsZero() {
						s.triggerConditions[sceneID][i].DurationStartTime = time.Now()
						continue
					}

					// 持续时长校验
					duration := time.Now().Sub(startTime)
					if int64(duration.Seconds()) < condition.Duration {
						continue
					}

					// 重置时间
					s.triggerConditions[sceneID][i].DurationStartTime = time.Now()
				case false: // 不处理持续条件时间
					if condition.Duration > 0 {
						continue
					}

					// 条件验证
					checkResult := s.checkConditionValue(condition, pointData.Value)
					if checkResult != nil { // 不满足条件
						continue
					}
				}

				// 触发场景
				go func(linkEdgeId string) {
					helper.Logger.Info("trigger linkEdge", zap.String("id", linkEdgeId))
					e := s.TriggerLinkEdge(linkEdgeId)
					if e != nil {
						helper.Logger.Error("trigger linkEdge error", zap.String("id", linkEdgeId), zap.Error(e))
					}
				}(sceneID)
				helper.Logger.Debug("check linkEdge condition success,break", zap.String("id", sceneID))
			}
		}
	}
}
