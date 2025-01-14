package linkage

import (
	"slices"
	"time"
)

// Config 场景配置
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
	// Triggers 触发器
	Triggers []Trigger `json:"triggers"`
	// Conditions 执行条件
	Conditions []Condition `json:"conditions"`
	// Actions 执行动作
	Actions []Action `json:"actions"`
	// LastExecuteTime 最后执行时间
	LastExecuteTime time.Time `json:"lastExecuteTime"`
}

// ContainsTag 判断场景是否包含指定标签
func (c *Config) ContainsTag(tag string) bool {
	return slices.Contains(c.Tags, tag)
}
