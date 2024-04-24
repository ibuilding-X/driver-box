package linkedge

import (
	bytes2 "bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"github.com/ibuilding-x/driver-box/driverbox/restful"
	"github.com/ibuilding-x/driver-box/driverbox/restful/route"
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	LinkConfigPath = "./config/linkedge" // 场景联动配置路径
)

// ActionListIsEmptyErr action 列表为空错误
var ActionListIsEmptyErr = errors.New("linkEdge action list cannot be empty")

type service struct {
	// 场景联动配置缓存
	configs map[string]ModelConfig
	// 定时任务
	schedules map[string]*cron.Cron
	//点位触发器
	triggerConditions map[string][]pointCondition

	envConfig EnvConfig
}

func (linkedge *service) NewService() error {
	// 创建联动场景
	restful.HandleFunc(route.LinkEdgeCreate, func(request *http.Request) (any, error) {
		data, err := readBody(request)
		if err != nil {
			err = fmt.Errorf("Incoming reading ignored. Unable to read request body: %s", err.Error())
			helper.Logger.Error(err.Error())
			return false, err
		}
		err = linkedge.Create(data)
		if err != nil {
			err = fmt.Errorf("create linkEdge error: %s", err.Error())
			helper.Logger.Error(err.Error())
			return false, err
		}
		return true, nil
	})

	// 预览联动场景
	restful.HandleFunc(route.LinkEdgeTryTrigger, func(request *http.Request) (any, error) {
		// 读取请求参数
		data, err := readBody(request)
		if err != nil {
			return false, err
		}
		// 解析数据
		var config ModelConfig
		err = json.Unmarshal(data, &config)
		if err != nil {
			return false, err
		}
		// 执行
		err = linkedge.triggerLinkEdge("", 0, config)
		if err != nil {
			err = fmt.Errorf("preview linkEdge error: %s", err.Error())
			helper.Logger.Error(err.Error())
			return false, err
		}
		return true, nil
	})

	//删除联动场景
	restful.HandleFunc(route.LinkEdgeDelete, func(request *http.Request) (any, error) {
		err := linkedge.Delete(request.FormValue("id"))
		return err != nil, err
	})

	//触发联动场景
	restful.HandleFunc(route.LinkEdgeTrigger, func(request *http.Request) (any, error) {
		helper.Logger.Info(fmt.Sprintf("trigger linkEdge:%s from: %s", request.FormValue("id"), request.FormValue("source")))
		err := linkedge.TriggerLinkEdge(request.FormValue("id"))
		return err != nil, err
	})

	//查看场景联动
	restful.HandleFunc(route.LinkEdgeGet, func(request *http.Request) (any, error) {
		helper.Logger.Info(fmt.Sprintf("get linkEdge:%s", request.FormValue("id")))
		return linkedge.getLinkEdge(request.FormValue("id"))
	})

	// 查看场景联动列表
	restful.HandleFunc(route.LinkEdgeList, func(request *http.Request) (any, error) {
		// 获取查询参数
		tag := request.URL.Query().Get("tag")
		return linkedge.GetList(tag)
	})

	restful.HandleFunc(route.LinkEdgeUpdate, func(request *http.Request) (any, error) {
		body, err := readBody(request)
		if err != nil {
			return false, err
		}
		var model ModelConfig
		err = json.Unmarshal(body, &model)
		if err != nil {
			return false, err
		}
		_, err = linkedge.getLinkEdge(model.Id)
		if err != nil {
			return false, err
		}
		err = linkedge.Update(body)
		if err != nil {
			return false, err
		} else {
			return true, nil
		}
	})

	//更新场景联动状态
	restful.HandleFunc(route.LinkEdgeStatus, func(request *http.Request) (any, error) {
		helper.Logger.Info(fmt.Sprintf("get linkEdge:%s", request.FormValue("id")))
		config, err := linkedge.getLinkEdge(request.FormValue("id"))
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
		if err = linkedge.Update(bf.Bytes()); err == nil {
			return true, nil
		} else {
			return false, fmt.Errorf("update %v error: %v", config, err)
		}
	})

	// 获取最后一次执行的场景信息
	restful.HandleFunc(route.LinkEdgeGetLast, func(r *http.Request) (any, error) {
		return linkedge.GetLast()
	})

	//启动场景联动
	configs, e := linkedge.GetList()
	if e != nil {
		return e
	}
	for _, config := range configs {
		e = linkedge.registerTrigger(config.Id)
		if e != nil {
			return e
		}
	}

	return nil
}

// Create 创建场景联动规则
func (linkEdge *service) Create(bytes []byte) error {
	var model ModelConfig
	e := json.Unmarshal(bytes, &model)
	if e != nil {
		return e
	}
	if _, exists := linkEdge.configs[model.Id]; exists {
		return errors.New("linkEdge id is exists!")
	}

	// 空 Action 校验
	if len(model.Action) <= 0 {
		return ActionListIsEmptyErr
	}

	//持久化
	file := path.Join(linkEdge.envConfig.ConfigPath, model.Id+".json")
	e = os.WriteFile(file, bytes, 0666)
	if e != nil {
		return e
	}

	//启动场景联动
	e = linkEdge.registerTrigger(model.Id)
	if e != nil {
		//注册触发器存在异常,清理脏数据
		_ = linkEdge.Delete(model.Id)
		return e
	}
	return nil
}

// 注册触发器
func (linkEdge *service) registerTrigger(id string) error {
	model, e := linkEdge.getLinkEdge(id)
	if e != nil {
		return e
	}
	//注册触发器
	for _, d := range model.Trigger {
		bytes, e := json.Marshal(d)
		if e != nil {
			return e
		}
		var baseTrigger baseTrigger
		e = json.Unmarshal(bytes, &baseTrigger)
		if e != nil {
			return e
		}
		switch baseTrigger.Type {
		case TriggerTypeSchedule:
			var scheduleTrigger scheduleTrigger
			e = json.Unmarshal(bytes, &scheduleTrigger)
			if e != nil {
				return e
			}

			schedule, exists := linkEdge.schedules[model.Id]
			if !exists {
				schedule = cron.New()
				linkEdge.schedules[model.Id] = schedule
				schedule.Start()
			}
			_, e = schedule.AddFunc(scheduleTrigger.Cron, func() {
				linkEdge.TriggerLinkEdge(model.Id)
			})
			if e != nil {
				return e
			} else {
				helper.Logger.Info(fmt.Sprintf("add schedule trigger:%v", scheduleTrigger.Cron))
			}
			break
		case TriggerTypeDevicePoint:
			//注册eKuiper监听设备点位状态
			var devicePointTrigger devicePointTrigger
			e = json.Unmarshal(bytes, &devicePointTrigger)
			if e != nil {
				return e
			}
			if len(devicePointTrigger.DeviceSn) == 0 || len(devicePointTrigger.DevicePoint) == 0 || len(devicePointTrigger.Condition) == 0 || len(devicePointTrigger.Value) == 0 {
				return errors.New("invalid trigger:" + string(bytes))
			}

			//定义触发器
			triggers, _ := linkEdge.triggerConditions[id]
			triggers = append(triggers, devicePointTrigger.pointCondition)
			linkEdge.triggerConditions[id] = triggers
			break
		case TriggerTypeDeviceEvent:
			break
		default:
			helper.Logger.Error(fmt.Sprintf("unsupport trigger type:%s", string(bytes)))
		}
	}
	return nil
}

// Delete 删除场景联动规则
func (linkEdge *service) Delete(id string) error {
	if len(id) == 0 {
		return errors.New("id is nil")
	}
	helper.Logger.Info(fmt.Sprintf("delete linkEdge:%v", id))

	//删除配置
	delete(linkEdge.configs, id)
	delete(linkEdge.triggerConditions, id)
	file := path.Join(linkEdge.envConfig.ConfigPath, id+".json")
	e := os.Remove(file)
	if e != nil {
		return e
	}

	// 清理当前场景ID的所有时间表触发器
	helper.Logger.Info(fmt.Sprintf("clear schedule trigger..."))
	s, exists := linkEdge.schedules[id]
	if exists {
		s.Stop()
		delete(linkEdge.schedules, id)
	}
	return nil
}

// UpdateLinkEdgeStatus 调整联动规则状态,用于启停控制
func (linkEdge *service) Update(bytes []byte) error {
	var model ModelConfig
	e := json.Unmarshal(bytes, &model)
	if e != nil {
		return e
	}
	// action 为空校验
	if len(model.Action) <= 0 {
		return ActionListIsEmptyErr
	}
	e = linkEdge.Delete(model.Id)
	if e != nil {
		return e
	}
	return linkEdge.Create(bytes)
}

// TriggerLinkEdge 触发场景联动规则
// id: 场景联动ID
// source: 场景触发来源
func (linkEdge *service) TriggerLinkEdge(id string) error {
	//记录场景执行记录
	e := linkEdge.triggerLinkEdge(id, 0)
	if e != nil {
		helper.Logger.Error(fmt.Sprintf("linkEdge:%s trigger", e.Error()))
		return e
	}
	//缓存场景联动执行时间
	config, e := linkEdge.getLinkEdge(id)
	if e == nil {
		config.executeTime = time.Now()
		linkEdge.configs[id] = config
	}
	return e
}

// depth:联动深度
func (linkEdge *service) triggerLinkEdge(id string, depth int, conf ...ModelConfig) error {
	if depth > 10 {
		return errors.New("execute level is too deep, max deep:" + strconv.Itoa(depth))
	}
	var config ModelConfig
	var e error
	if len(conf) > 0 {
		config = conf[0]
	} else {
		// 核对触发器
		config, e = linkEdge.getLinkEdge(id)
		if e != nil {
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
		if time.Now().Add(-time.Duration(config.SilentPeriod) * time.Second).Before(config.executeTime) {
			return errors.New("execute frequency is too high")
		}
	}
	//校验执行条件
	e = linkEdge.checkConditions(config.Condition)
	if e != nil {
		return errors.New("check condition error: " + e.Error())
	}

	sucCount := 0
	//执行动作
	for _, action := range config.Action {
		bytes, e := json.Marshal(action)
		helper.Logger.Info(fmt.Sprintf("will doAction: %s", string(bytes)))
		if e != nil {
			helper.Logger.Error(fmt.Sprintf("execute linkEdge:%s action error:%s", id, e.Error()))
			continue
		}
		var baseAction baseAction
		e = json.Unmarshal(bytes, &baseAction)
		if e != nil {
			helper.Logger.Error(fmt.Sprintf("execute linkEdge:%s action error:%s", id, e.Error()))
			continue
		}

		//判断执行动作是否匹配条件
		e = linkEdge.checkConditions(baseAction.Condition)
		if e != nil {
			sucCount++
			continue
		}

		switch baseAction.Type {
		// 设置设备点位
		case ActionTypeDevicePoint:
			var devicePointAction devicePointAction
			e = json.Unmarshal(bytes, &devicePointAction)
			if e != nil {
				helper.Logger.Error(fmt.Sprintf("execute linkEdge:%s action error:%s", id, e.Error()))
				continue
			}
			err := core.SendSinglePoint(devicePointAction.DeviceSn, plugin.WriteMode, plugin.PointData{
				PointName: devicePointAction.DevicePoint,
				Value:     devicePointAction.Value,
			})
			if err != nil {
				helper.Logger.Error("execute linkEdge error", zap.String("linkEdge", id),
					zap.String("pointName", devicePointAction.DevicePoint), zap.String("pointValue", devicePointAction.Value), zap.Error(err))
			} else {
				sucCount++
				helper.Logger.Info(fmt.Sprintf("execute linkEdge:%s action", id))
			}
			break
			//触发下一个场景联动
		case ActionTypeLinkEdge:
			var linkEdgeAction linkEdgeAction
			e = json.Unmarshal(bytes, &linkEdgeAction)
			if e != nil {
				helper.Logger.Error(fmt.Sprintf("execute linkEdge:%s action error:%s", id, e.Error()))
			} else {
				sucCount++
				go linkEdge.triggerLinkEdge(linkEdgeAction.Id, depth+1)
			}
			break
		default:
			helper.Logger.Error(fmt.Sprintf("unsupport action:%s", string(bytes)))
		}

		//场景执行后休眠指定时长
		if len(baseAction.Sleep) > 0 {
			d, err := time.ParseDuration(baseAction.Sleep)
			if err == nil {
				time.Sleep(d)
			}
		}
	}
	if id != "" {
		// value:全部成功\部分成功\全部失败
		if sucCount == len(config.Action) {
			helper.TriggerEvents(EventCodeLinkEdgeTrigger, id, LinkEdgeExecuteResultAllSuccess)
		} else if sucCount == 0 {
			helper.TriggerEvents(EventCodeLinkEdgeTrigger, id, LinkEdgeExecuteResultAllFail)
		} else {
			helper.TriggerEvents(EventCodeLinkEdgeTrigger, id, LinkEdgeExecuteResultPartSuccess)
		}
	}

	return nil
}

func (linkEdge *service) checkConditions(conditions []interface{}) error {
	//优先执行点位持续时间条件校验
	err := linkEdge.checkListTimeCondition(conditions)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for _, c := range conditions {
		helper.Logger.Info(fmt.Sprintf("check condition:%v", c))
		bytes, e := json.Marshal(c)
		if e != nil {
			return e
		}
		var baseCondition baseCondition
		err := json.Unmarshal(bytes, &baseCondition)
		if err != nil {
			return err
		}
		if baseCondition.Type == ConditionLastTime {
			continue
		}
		switch baseCondition.Type {
		case ConditionTypeDevicePoint:
			//注册eKuiper监听设备点位状态
			var condition devicePointCondition
			err = json.Unmarshal(bytes, &condition)
			if err != nil {
				return err
			}
			if len(condition.DeviceSn) == 0 || len(condition.DevicePoint) == 0 || len(condition.Condition) == 0 || len(condition.Value) == 0 {
				return errors.New("invalid trigger:" + string(bytes))
			}
			pointValue, err := helper.DeviceShadow.GetDevicePoint(condition.DeviceSn, condition.DevicePoint)
			if err != nil {
				return fmt.Errorf("get device:%v point:%v value error:%v", condition.DeviceSn, condition.DevicePoint, err)
			}
			//helper.Logger.Info(fmt.Sprintf("point value:%s", point))
			err = linkEdge.checkConditionValue(condition.pointCondition, pointValue)
			if err != nil {
				return err
			}
		case ConditionExecuteTime:
			var condition executeTimeCondition
			err = json.Unmarshal(bytes, &condition)
			if err != nil {
				return err
			}
			if condition.Begin > now {
				return errors.New("execution time has not started")
			}
			if condition.End < now {
				return errors.New("execution time has expired")
			}
		case ConditionDateInterval:
			var condition dateIntervalCondition
			if err = json.Unmarshal(bytes, &condition); err != nil {
				return err
			}
			if condition.BeginDate == "" || condition.EndDate == "" {
				return nil
			}
			yearDay := time.Now().YearDay()
			begin, err := linkEdge.parseDate(condition.BeginDate)
			if err != nil {
				return errors.New("execution begin date parse error")
			}
			if yearDay < begin.YearDay() {
				return errors.New("execution time has not started")
			}
			end, err := linkEdge.parseDate(condition.EndDate)
			if err != nil {
				return errors.New("execution end date parse error")
			}
			if yearDay > end.YearDay() {
				return errors.New("execution time has expired")
			}
		case ConditionYears:
			var condition yearsCondition
			if err = json.Unmarshal(bytes, &condition); err != nil {
				return err
			}
			if !condition.Verify(time.Now().Year()) {
				return errors.New("mismatch years condition")
			}
		case ConditionMonths:
			var condition monthsCondition
			if err = json.Unmarshal(bytes, &condition); err != nil {
				return err
			}
			if !condition.Verify(int(time.Now().Month())) {
				return errors.New("mismatch months condition")
			}
		case ConditionDays:
			var condition daysCondition
			if err = json.Unmarshal(bytes, &condition); err != nil {
				return err
			}
			if !condition.Verify(time.Now().Day()) {
				return errors.New("mismatch days condition")
			}
		case ConditionWeeks:
			var condition weeksCondition
			if err = json.Unmarshal(bytes, &condition); err != nil {
				return err
			}
			if !condition.Verify(int(time.Now().Weekday())) {
				return errors.New("mismatch weeks condition")
			}
		case ConditionTimes:
			var condition timesCondition
			if err = json.Unmarshal(bytes, &condition); err != nil {
				return err
			}
			if !condition.Verify(time.Now()) {
				return errors.New("mismatch times condition")
			}
		}
	}
	return nil
}

func (linkEdge *service) checkListTimeCondition(conditions []interface{}) error {
	//return errors.New("功能未迁移...")
	return nil
}

func (linkEdge *service) checkConditionValue(condition pointCondition, pointValue interface{}) error {
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

func (linkEdge *service) getLinkEdge(id string) (ModelConfig, error) {
	config, exists := linkEdge.configs[id]
	if exists {
		return config, nil
	}

	config = ModelConfig{}
	fileName := filepath.Join(linkEdge.envConfig.ConfigPath, id+".json")
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
	linkEdge.configs[id] = config
	return config, err
}

func (linkEdge *service) GetList(tag ...string) ([]ModelConfig, error) {
	files := make([]string, 0)
	//若目录不存在，则自动创建
	_, err := os.Stat(linkEdge.envConfig.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(linkEdge.envConfig.ConfigPath, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	filepath.Walk(linkEdge.envConfig.ConfigPath, func(path string, d fs.FileInfo, err error) error {
		if !d.IsDir() {
			files = append(files, d.Name())
		}
		return nil
	})

	var configs []ModelConfig

	for _, key := range files {
		id := strings.TrimSuffix(key, ".json")
		config, err := linkEdge.getLinkEdge(id)
		if err != nil {
			return configs, err
		} else {
			if len(tag) > 0 && tag[0] != "" {
				if config.hasTag(tag[0]) {
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
func (linkEdge *service) GetLast() (c ModelConfig, err error) {
	defaultTime := time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
	configs, err := linkEdge.GetList()
	if err != nil {
		return
	}
	for _, config := range configs {
		if config.executeTime.After(defaultTime) || config.executeTime.Equal(defaultTime) {
			defaultTime = config.executeTime
			c = config
		}
	}
	// 判断执行时间，若执行时间为空，则返回空
	if c.executeTime.IsZero() {
		return ModelConfig{}, nil
	}
	return
}

func readBody(request *http.Request) ([]byte, error) {
	defer request.Body.Close()
	body, err := ioutil.ReadAll(request.Body)
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
func (linkEdge *service) parseDate(d string) (time.Time, error) {
	year := time.Now().Year()
	if d == "02-29" && (year%4 != 0) {
		d = "02-28"
	}
	return time.Parse("2006-01-02", fmt.Sprintf("%d-%s", year, d))
}

// Preview 预览场景
// 提示：不真实创建场景，仅看执行效果使用
func (linkEdge *service) Preview(config ModelConfig) error {
	// 记录场景执行记录
	return linkEdge.triggerLinkEdge("", 0, config)
}
