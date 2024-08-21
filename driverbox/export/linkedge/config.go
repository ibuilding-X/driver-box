package linkedge

import "time"

type Config struct {
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
	Trigger []Trigger `json:"trigger"`
	// 场景联动的执行条件
	Condition []Condition `json:"condition"`
	// 执行动作
	Action []Action `json:"action"`
	//上一次执行时间
	executeTime time.Time
}
