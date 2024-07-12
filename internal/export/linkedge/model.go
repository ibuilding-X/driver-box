package linkedge

import (
	"encoding/json"
	"time"
)

const (
	// TriggerTypeSchedule 触发器：时间表
	TriggerTypeSchedule triggerType = "schedule"
	// TriggerTypeDevicePoint 触发器：设备点位
	TriggerTypeDevicePoint triggerType = "devicePoint"
	// TriggerTypeDeviceEvent 触发器：设备事件
	TriggerTypeDeviceEvent triggerType = "deviceEvent"

	// ConditionTypeDevicePoint 执行条件：设备点位
	ConditionTypeDevicePoint conditionType = "devicePoint"
	//执行条件：有些运行时间段
	ConditionExecuteTime conditionType = "executeTime"
	// ConditionLastTime 执行条件 持续时间
	ConditionLastTime conditionType = "lastTime"
	// ConditionDateInterval 日期间隔，示例：01-01 ～ 05-01
	// Deprecated: 已废弃，请拆分为多个条件，例如：年（数组）、月（数组）、日（数组）、周（数组）、时间段（数组）
	ConditionDateInterval conditionType = "dateInterval"
	// ConditionYears 执行条件-年（数组）
	ConditionYears conditionType = "years"
	// ConditionMonths 执行条件-月（数组）
	ConditionMonths conditionType = "months"
	// ConditionDays 执行条件-日（数组）
	ConditionDays conditionType = "days"
	// ConditionWeeks 执行条件-周（数组）
	ConditionWeeks conditionType = "weeks"
	// ConditionTimes 执行条件-时间段（数组）
	ConditionTimes conditionType = "times"

	// 执行类型：设置设备点位
	ActionTypeDevicePoint ActionType = "devicePoint"
	// 执行类型：触发场景联动
	ActionTypeLinkEdge ActionType = "linkEdge"

	ConditionEq conditionSymbol = "="
	ConditionNe conditionSymbol = "!="
	ConditionGt conditionSymbol = ">"
	ConditionGe conditionSymbol = ">="
	ConditionLt conditionSymbol = "<"
	ConditionLe conditionSymbol = "<="

	//场景联动执行结果：全部成功、部分成功、全部失败
	LinkEdgeExecuteResultAllSuccess  = "success"
	LinkEdgeExecuteResultPartSuccess = "partSuccess"
	LinkEdgeExecuteResultAllFail     = "fail"
)

// 触发器类型
type triggerType string

// 执行条件类型
type conditionType string
type ActionType string
type httpMethod string

// 条件符号
type conditionSymbol string

// 场景联动配置模型
type ModelConfig struct {
	//场景ID
	Id string `json:"id,omitempty"`
	//是否可用
	Enable bool `json:"enable"`
	//场景名称
	Name string `json:"name"`
	//场景标签
	Tags []string `json:"tags"`
	// 场景描述
	Description string `json:"description"`
	// 静默期,单位：秒
	SilentPeriod int64 `json:"silentPeriod"`
	// 触发器
	Trigger []interface{} `json:"trigger"`
	// 场景联动的执行条件
	Condition []interface{} `json:"condition"`
	// 执行动作
	Action []interface{} `json:"action"`

	//上一次执行时间
	executeTime time.Time
}

func (mc ModelConfig) hasTag(tag string) bool {
	for i, _ := range mc.Tags {
		if tag == mc.Tags[i] {
			return true
		}
	}
	return false
}

// 获取该场景最近一次执行时间
func (mc ModelConfig) GetExecuteTime() time.Time {
	return mc.executeTime
}

type baseTrigger struct {
	Type triggerType `json:"type"`
}

type pointCondition struct {
	// 场景联动中用 devSn 表示设备ID
	DeviceId string `json:"devSn"`
	//云端定义的设备点位 point 在边缘侧用ResourceName表示
	DevicePoint string `json:"point"`
	//条件模式：== != > < 等
	Condition conditionSymbol `json:"condition"`
	// 条件值
	Value string `json:"value"`
}

type devicePointTrigger struct {
	baseTrigger
	pointCondition
}

// 时间表触发器
type scheduleTrigger struct {
	baseTrigger
	// cron表达式
	Cron string `json:"cron"`
}

type baseCondition struct {
	Type conditionType `json:"type"`
}
type devicePointCondition struct {
	baseCondition
	pointCondition
}

// 执行条件：有效执行时间段
type executeTimeCondition struct {
	baseCondition
	Begin int64 `json:"begin"`
	End   int64 `json:"end"`
}

type lastTimeCondition struct {
	devicePointCondition
	LastTime int64 `json:"lastTime"`
}

// dateIntervalCondition 条件 - 日期间隔
// 日期格式：01-02
type dateIntervalCondition struct {
	baseCondition
	BeginDate string `json:"begin_date"`
	EndDate   string `json:"end_date"`
}

// yearsCondition 条件 - 年份
// demo: 2021-2025 表示2021年到2025年
type yearsCondition struct {
	baseCondition
	Years []int `json:"years"`
}

// monthsCondition 条件 - 月份
// demo: 1-12 表示1月到12月
type monthsCondition struct {
	baseCondition
	Months []int `json:"months"`
}

// daysCondition 条件 - 日期
// demo: 1-31 表示1号到31号
type daysCondition struct {
	baseCondition
	Days []int `json:"days"`
}

// weeksCondition 条件 - 星期
// demo: 1-7 表示星期一到星期日
type weeksCondition struct {
	baseCondition
	Weeks []int `json:"weeks"`
}

// timesCondition 条件 - 时间段
// 示例：begin_time: "08:00", end_time: "18:00"
type timesCondition struct {
	baseCondition
	BeginTime string `json:"begin_time"`
	EndTime   string `json:"end_time"`
}

type baseAction struct {
	Type ActionType `json:"type"`
	// Action的执行条件
	Condition []interface{} `json:"condition"`
	// 附加属性（例如：供前端存储设备组信息）
	Attrs map[string]any `json:"attrs"`

	// 执行后休眠时长
	Sleep string `json:"sleep"`
}

type devicePointAction struct {
	baseAction
	//设备名称
	DeviceId string `json:"devSn"`
	// 点位名
	DevicePoint string `json:"point"`
	// 点位值
	Value string `json:"value"`
}

type linkEdgeAction struct {
	baseCondition
	// 场景联动ID
	Id string `json:"id"`
}

type BaseResponse struct {
	Success   bool        `json:"success"`
	ErrorCode int         `json:"errorCode"`
	ErrorMsg  string      `json:"errorMsg"`
	Data      interface{} `json:"data"`
}

func ok(data interface{}) string {
	resp := &BaseResponse{
		Success: true,
		Data:    data,
	}
	bytes, _ := json.Marshal(resp)
	return string(bytes)
}

func fail(err error) string {
	resp := &BaseResponse{
		Success:   false,
		ErrorCode: 500,
		ErrorMsg:  err.Error(),
	}
	bytes, _ := json.Marshal(resp)
	return string(bytes)
}
