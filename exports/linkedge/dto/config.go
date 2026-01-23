package dto

import "time"

type Config struct {
	// ID 场景ID
	ID string `json:"id,omitempty"`
	// Enable 是否可用
	Enable bool `json:"enable"`
	// Name 场景名称
	Name string `json:"name"`
	// Tags 场景标签
	Tags []string `json:"tags"`
	// Description 场景描述
	Description string `json:"description"`
	// SilentPeriod 静默期,单位：秒
	SilentPeriod int64 `json:"silentPeriod"`
	// Trigger 触发器
	Trigger []Trigger `json:"trigger"`
	// Condition 执行条件
	Condition []Condition `json:"condition"`
	// Action 执行动作
	Action []Action `json:"action"`
	// ExecuteTime 最后执行时间
	ExecuteTime time.Time
}

func (c *Config) ExistTag(tag string) bool {
	for i, _ := range c.Tags {
		if tag == c.Tags[i] {
			return true
		}
	}

	return false
}
