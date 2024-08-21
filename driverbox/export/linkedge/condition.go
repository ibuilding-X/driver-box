package linkedge

// ConditionSymbol 条件符号
type ConditionSymbol string

// ConditionType 条件类型
type ConditionType string

const (
	ConditionEq ConditionSymbol = "="
	ConditionNe ConditionSymbol = "!="
	ConditionGt ConditionSymbol = ">"
	ConditionGe ConditionSymbol = ">="
	ConditionLt ConditionSymbol = "<"
	ConditionLe ConditionSymbol = "<="
)

const (
	// ConditionTypeDevicePoint 执行条件-设备点位
	ConditionTypeDevicePoint ConditionType = "devicePoint"
	// ConditionTypeExecuteTime 执行条件-有些运行时间段
	ConditionTypeExecuteTime ConditionType = "executeTime"
	// ConditionTypeLastTime 执行条件-持续时间
	ConditionTypeLastTime ConditionType = "lastTime"
	// ConditionTypeDateInterval 执行条件-日期间隔，示例：01-01 ～ 05-01
	// Deprecated: 已废弃，请拆分为多个条件，例如：年（数组）、月（数组）、日（数组）、周（数组）、时间段（数组）
	ConditionTypeDateInterval ConditionType = "dateInterval"
	// ConditionTypeYears 执行条件-年（数组）
	ConditionTypeYears ConditionType = "years"
	// ConditionTypeMonths 执行条件-月（数组）
	ConditionTypeMonths ConditionType = "months"
	// ConditionTypeDays 执行条件-日（数组）
	ConditionTypeDays ConditionType = "days"
	// ConditionTypeWeeks 执行条件-周（数组）
	ConditionTypeWeeks ConditionType = "weeks"
	// ConditionTypeTimes 执行条件-时间段（数组）
	ConditionTypeTimes ConditionType = "times"
)

// Condition 条件
type Condition struct {
	Type ConditionType `json:"type"`
	DevicePointCondition
	ExecuteTimeCondition
	LastTimeCondition
	DateIntervalCondition
	YearsCondition
	MonthsCondition
	DaysCondition
	WeeksCondition
	TimesCondition
}

// DevicePointCondition 设备点位条件
type DevicePointCondition struct {
	// DeviceID 设备 ID
	DeviceID string `json:"devSn"`
	// DevicePoint 点位名称
	DevicePoint string `json:"point"`
	// Condition 条件模式：== != > < 等
	Condition ConditionSymbol `json:"condition"`
	// Value 条件值
	Value string `json:"value"`
}

// ExecuteTimeCondition 有效执行时间段
type ExecuteTimeCondition struct {
	Begin int64 `json:"begin"`
	End   int64 `json:"end"`
}

// LastTimeCondition 持续时间条件
type LastTimeCondition struct {
	LastTime int64 `json:"lastTime"`
}

// DateIntervalCondition 日期间隔
// 日期格式：01-02
type DateIntervalCondition struct {
	BeginDate string `json:"begin_date"`
	EndDate   string `json:"end_date"`
}

// YearsCondition 年份
// demo: 2021-2025 表示2021年到2025年
type YearsCondition struct {
	Years []int `json:"years"`
}

// MonthsCondition 月份
// demo: 1-12 表示1月到12月
type MonthsCondition struct {
	Months []int `json:"months"`
}

// DaysCondition 日期
// demo: 1-31 表示1号到31号
type DaysCondition struct {
	Days []int `json:"days"`
}

// WeeksCondition 星期
// demo: 1-7 表示星期一到星期日
type WeeksCondition struct {
	Weeks []int `json:"weeks"`
}

// TimesCondition 时间段
// 示例：begin_time: "08:00", end_time: "18:00"
type TimesCondition struct {
	BeginTime string `json:"begin_time"`
	EndTime   string `json:"end_time"`
}
