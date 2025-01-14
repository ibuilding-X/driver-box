package linkage

// ConditionSymbol 条件符号
type ConditionSymbol string

// ConditionType 条件类型
type ConditionType string

const (
	ConditionSymbolEq ConditionSymbol = "="
	ConditionSymbolNe ConditionSymbol = "!="
	ConditionSymbolGt ConditionSymbol = ">"
	ConditionSymbolGe ConditionSymbol = ">="
	ConditionSymbolLt ConditionSymbol = "<"
	ConditionSymbolLe ConditionSymbol = "<="
)

const (
	// ConditionTypeDevicePoint 执行条件-设备点位
	ConditionTypeDevicePoint ConditionType = "devicePoint"
	// ConditionTypeExecuteTime 执行条件-有些运行时间段
	ConditionTypeExecuteTime ConditionType = "executeTime"
	// ConditionTypeLastTime 执行条件-持续时间
	ConditionTypeLastTime ConditionType = "lastTime"
	// ConditionTypeDateInterval 执行条件-日期间隔，示例：01-01 ～ 05-01
	ConditionTypeDateInterval ConditionType = "dateInterval"
	// ConditionTypeYears 执行条件-年（数组）
	ConditionTypeYears ConditionType = "years"
	// ConditionTypeMonths 执行条件-月（数组）
	ConditionTypeMonths ConditionType = "months"
	// ConditionTypeDays 执行条件-日（数组）
	ConditionTypeDays ConditionType = "days"
	// ConditionTypeWeeks 执行条件-周（数组）
	ConditionTypeWeeks ConditionType = "weeks"
	// ConditionTypeTimeInterval 执行条件-时间段（数组）
	ConditionTypeTimeInterval ConditionType = "timeInterval"
)

// Condition 条件
type Condition struct {
	Type ConditionType `json:"type" validate:"required,oneof=devicePoint executeTime lastTime dateInterval years months days weeks timeInterval"`
	DevicePointCondition
	ExecuteTimeCondition
	LastTimeCondition
	DateIntervalCondition
	YearsCondition
	MonthsCondition
	DaysCondition
	WeeksCondition
	TimeIntervalCondition
}

// DevicePointCondition 设备点位条件
type DevicePointCondition struct {
	// DeviceID 设备 ID
	DeviceID string `json:"devSn" validate:"omitempty"`
	// DevicePoint 点位名称
	DevicePoint string `json:"point" validate:"omitempty"`
	// Condition 条件模式：== != > < 等
	Condition ConditionSymbol `json:"condition" validate:"omitempty,oneof='=' '!=' '>' '>=' '<' '<='"`
	// Value 条件值
	Value string `json:"value" validate:"omitempty"`
}

// ExecuteTimeCondition 有效执行时间段
type ExecuteTimeCondition struct {
	Begin int64 `json:"begin" validate:"omitempty"`
	End   int64 `json:"end" validate:"omitempty"`
}

// LastTimeCondition 持续时间条件
type LastTimeCondition struct {
	LastTime int64 `json:"lastTime"`
}

// DateIntervalCondition 日期间隔
// 日期格式：01-02
type DateIntervalCondition struct {
	BeginDate string `json:"begin_date" validate:"omitempty,datetime=01-02"`
	EndDate   string `json:"end_date" validate:"omitempty,datetime=01-02"`
}

// YearsCondition 年份
// demo: 2021-2025 表示2021年到2025年
type YearsCondition struct {
	Years []int `json:"years" validate:"omitempty"`
}

// MonthsCondition 月份
// demo: 1-12 表示1月到12月
type MonthsCondition struct {
	Months []int `json:"months" validate:"omitempty"`
}

// DaysCondition 日期
// demo: 1-31 表示1号到31号
type DaysCondition struct {
	Days []int `json:"days" validate:"omitempty"`
}

// WeeksCondition 星期
// demo: 1-7 表示星期一到星期日
type WeeksCondition struct {
	Weeks []int `json:"weeks" validate:"omitempty"`
}

// TimeIntervalCondition 时间段
// 示例：begin_time: "08:00", end_time: "18:00"
type TimeIntervalCondition struct {
	BeginTime string `json:"begin_time" validate:"omitempty,datetime=15:04"`
	EndTime   string `json:"end_time" validate:"omitempty,datetime=15:04"`
}
