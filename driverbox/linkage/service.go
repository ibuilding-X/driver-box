package linkage

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"strconv"
	"sync"
	"time"
)

type Callback func(result ExecutionResult)

type ExecutionResult struct {
	ID      string                `json:"id"`
	Success bool                  `json:"success"`
	Details ExecutionResultDetail `json:"details"`
}

type ExecutionResultDetail struct {
	Device
	ErrorMessage string `json:"error_message"`
}

type service struct {
	// configs 配置列表
	configs sync.Map
	// schedules 定时任务列表
	schedules map[string]*cron.Cron
	// triggerConditions点位触发器
	triggerConditions map[string][]DevicePointCondition
	// deviceReadWriter 设备读写器
	deviceReadWriter deviceReadWriter
	// execResultCallback 执行结果回调函数
	execResultCallback Callback
}

func (s *service) Add(config Config) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if _, ok := s.configs.Load(config.ID); ok {
		return ErrAlreadyExist
	}

	if len(config.Actions) == 0 {
		return ErrActionEmpty
	}

	return s.add(config)
}

func (s *service) Update(config Config) error {
	if _, ok := s.configs.Load(config.ID); !ok {
		return ErrNotFound
	}

	s.delete(config.ID)
	return s.add(config)
}

func (s *service) Delete(id string) error {
	if _, ok := s.configs.Load(id); !ok {
		return ErrNotFound
	}

	s.delete(id)
	return nil
}

func (s *service) Trigger(id string) error {
	configAny, ok := s.configs.Load(id)
	if !ok {
		return ErrNotFound
	}

	if err := s.trigger(id, 0); err != nil {
		return err
	}

	config, _ := configAny.(Config)
	config.LastExecuteTime = time.Now()
	s.configs.Store(id, config)
	return nil
}

func (s *service) All() []Config {
	var configs []Config
	s.configs.Range(func(key, value any) bool {
		config, _ := value.(Config)
		configs = append(configs, config)
		return true
	})

	return configs
}

func (s *service) QueryByID(id string) (Config, error) {
	configAny, ok := s.configs.Load(id)
	if !ok {
		return Config{}, ErrNotFound
	}

	config, _ := configAny.(Config)
	return config, nil
}

func (s *service) QueryByTag(tag string) ([]Config, error) {
	var configs []Config
	s.configs.Range(func(key, value any) bool {
		config, _ := value.(Config)
		if config.ContainsTag(tag) {
			configs = append(configs, config)
		}
		return true
	})
	return configs, nil
}

func (s *service) QueryByLastTrigger() (Config, error) {
	var lastExecuteTime time.Time
	var c Config

	s.configs.Range(func(key, value any) bool {
		config, _ := value.(Config)
		if config.LastExecuteTime.After(lastExecuteTime) {
			lastExecuteTime = config.LastExecuteTime
			c = config
		}
		return true
	})

	if c.LastExecuteTime.IsZero() {
		return Config{}, nil
	}
	return c, nil
}

func (s *service) DevicePointsBus(device Device) {
	for _, p := range device.Points {
		for id, conditions := range s.triggerConditions {
			for _, condition := range conditions {
				if condition.DeviceID != device.DeviceID || condition.DevicePoint != p.Point {
					continue
				}

				// 同一个场景联动任意触发条件符合即可
				if s.checkConditionValue(condition, p.Value) == nil {
					go func(linkEdgeId string) {
						_ = s.Trigger(linkEdgeId)
					}(id)
					break
				}
			}

		}
	}
}

func (s *service) add(config Config) error {
	s.configs.Store(config.ID, config)
	return s.registerTrigger(config)
}

func (s *service) delete(id string) {
	// 删除配置
	s.configs.Delete(id)
	// 删除触发条件
	delete(s.triggerConditions, id)
	// 删除定时任务
	if schedule, ok := s.schedules[id]; ok {
		schedule.Stop()
		delete(s.schedules, id)
	}
}

// registerTrigger 注册触发器
func (s *service) registerTrigger(config Config) error {
	for _, trigger := range config.Triggers {
		switch trigger.Type {
		case TriggerTypeSchedule:
			schedule, ok := s.schedules[config.ID]
			if !ok {
				schedule = cron.New()
				s.schedules[config.ID] = schedule
				schedule.Start()
			}
			_, err := schedule.AddFunc(trigger.Cron, func() {
				_ = s.Trigger(config.ID)
			})
			if err != nil {
				return err
			}
		case TriggerTypeDevicePoint:
			if len(trigger.DeviceID) == 0 ||
				len(trigger.DevicePoint) == 0 ||
				len(trigger.Condition) == 0 ||
				len(trigger.Value) == 0 {
				bs, _ := json.Marshal(trigger.DevicePointTrigger)
				return fmt.Errorf("invalid trigger: %s", bs)
			}
		case TriggerTypeDeviceEvent:
			// todo something
		default:
			return ErrUnknownTriggerType
		}
	}

	return nil
}

func (s *service) trigger(id string, depth int) error {
	if depth > 10 {
		return ErrTooDeep
	}

	configAny, ok := s.configs.Load(id)
	if !ok {
		return ErrNotFound
	}
	config, _ := configAny.(Config)

	if config.Enable == false {
		return nil
	}

	if config.SilentPeriod > 0 {
		if time.Now().Sub(config.LastExecuteTime) < time.Duration(config.SilentPeriod)*time.Second {
			return ErrExecTooFast
		}
	}

	if err := s.checkConditions(config.Conditions); err != nil {
		return err
	}

	// 组合相同设备的点位 action
	actions := make(map[string][]DevicePoint)
	sucCount := 0
	// 执行动作
	for _, action := range config.Actions {
		//判断执行动作是否匹配条件
		e := s.checkConditions(action.Condition)
		if e != nil {
			sucCount++
			continue
		}

		switch action.Type {
		// 设置设备点位
		case ActionTypeDevicePoint:
			deviceID := action.DeviceID
			if _, ok := actions[deviceID]; !ok {
				actions[deviceID] = make([]DevicePoint, 0)
			}

			// 多点位设置
			if len(action.Points) != 0 {
				for _, point := range action.Points {
					actions[deviceID] = append(actions[deviceID], DevicePoint{
						Point: point.Point,
						Value: point.Value,
					})
				}
			}
		case ActionTypeLinkage:
			sucCount++
			go s.trigger(action.ID, depth+1)
		default:
			bytes, _ := json.Marshal(action)
			return fmt.Errorf("invalid action: %s", bytes)
		}

		// 场景执行后休眠指定时长
		if len(action.Sleep) > 0 {
			d, err := time.ParseDuration(action.Sleep)
			if err == nil {
				time.Sleep(d)
			}
		}
	}
	// 遍历执行 actions，按连接分组
	connectGroup := make(map[string]map[string][]DevicePoint)
	for deviceId, points := range actions {
		device, ok := helper.CoreCache.GetDevice(deviceId)
		if !ok {
			helper.Logger.Error("get device error", zap.String("deviceId", deviceId))
			continue
		}
		group, ok := connectGroup[device.ConnectionKey]
		if !ok {
			group = make(map[string][]DevicePoint)
			connectGroup[device.ConnectionKey] = group
		}
		group[deviceId] = points
	}
	var wg sync.WaitGroup
	for _, devices := range connectGroup {
		wg.Add(1)
		go func(ds map[string][]DevicePoint) {
			defer wg.Done()
			for deviceId, points := range ds {
				err := s.deviceReadWriter.Write(deviceId, points)
				if err != nil {
					//helper.Logger.Error("execute linkEdge error", zap.String("linkEdge", id),
					//	zap.String("deviceId", deviceId), zap.Any("points", points), zap.Error(err))
				} else {
					sucCount = sucCount + len(points)
					//helper.Logger.Info(fmt.Sprintf("execute linkEdge:%s action", id))
				}
			}
		}(devices)
	}
	wg.Wait()

	var result ExecutionResult
	result.ID = id
	if sucCount == len(config.Actions) {
		result.Success = true
	}
	s.execResultCallback(result)

	return nil
}

// parseDate 解析日期
func (s *service) parseDate(d string) (time.Time, error) {
	year := time.Now().Year()
	if d == "02-29" && (year%4 != 0) {
		d = "02-28"
	}
	return time.Parse("2006-01-02", fmt.Sprintf("%d-%s", year, d))
}

func (s *service) checkConditionValue(condition DevicePointCondition, pointValue interface{}) error {
	e := errors.New(fmt.Sprintf("condition check fail. expect %v%v%v ,actual value=%v", condition.DevicePoint, condition.Condition, condition.Value, pointValue))
	switch pointValue.(type) {
	case string:
		switch condition.Condition {
		case ConditionSymbolEq:
			if condition.Value != pointValue {
				return e
			}
			break
		case ConditionSymbolNe:
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
		case ConditionSymbolEq:
			if conditionValue != pointValue {
				return e
			}
			break
		case ConditionSymbolNe:
			if conditionValue == pointValue {
				return e
			}
			break
		case ConditionSymbolGt:
			if conditionValue >= pointValue {
				return e
			}
			break
		case ConditionSymbolGe:
			if conditionValue > pointValue {
				return e
			}
			break
		case ConditionSymbolLt:
			if conditionValue <= pointValue {
				return e
			}
			break
		case ConditionSymbolLe:
			if conditionValue < pointValue {
				return e
			}
			break
		}
	}

	return nil
}

func (s *service) checkConditions(conditions []Condition) error {
	now := time.Now().UnixMilli()

	for _, condition := range conditions {
		switch condition.Type {
		case ConditionTypeLastTime:
			if len(condition.DeviceID) == 0 ||
				len(condition.DevicePoint) == 0 ||
				len(condition.Condition) == 0 ||
				len(condition.Value) == 0 {
				bytes, _ := json.Marshal(condition.DevicePointCondition)
				return errors.New("invalid trigger:" + string(bytes))
			}
			pointValue, err := s.deviceReadWriter.Read(condition.DeviceID, condition.DevicePoint)
			if err != nil {
				return fmt.Errorf("get device:%v point:%v value error:%v", condition.DeviceID, condition.DevicePoint, err)
			}
			err = s.checkConditionValue(condition.DevicePointCondition, pointValue)
			if err != nil {
				return err
			}
		case ConditionTypeDevicePoint:
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
		case ConditionTypeTimeInterval:
			if !condition.TimeIntervalCondition.Verify(time.Now()) {
				return errors.New("mismatch times condition")
			}
		default:
			return ErrUnknownConditionType
		}
	}

	return nil
}
