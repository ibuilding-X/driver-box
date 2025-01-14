package linkage

import (
	"github.com/go-playground/validator/v10"
	"slices"
	"time"
)

var validate = validator.New()

// Config 场景配置
type Config struct {
	// ID 场景ID
	ID string `json:"id" validate:"required"`
	// Enable 是否可用
	Enable bool `json:"enable" validate:"required"`
	// Name 场景名称
	Name string `json:"name" validate:"required,max=255"`
	// Tags 场景标签
	Tags []string `json:"tags" validate:""`
	// Description 场景描述
	Description string `json:"description" validate:"omitempty,max=255"`
	// SilentPeriod 静默期,单位：秒
	SilentPeriod int64 `json:"silentPeriod" validate:""`
	// Triggers 触发器
	Triggers []Trigger `json:"triggers" validate:""`
	// Conditions 执行条件
	Conditions []Condition `json:"conditions" validate:""`
	// Actions 执行动作
	Actions []Action `json:"actions" validate:"required"`
	// LastExecuteTime 最后执行时间
	LastExecuteTime time.Time `json:"lastExecuteTime" validate:"required"`
}

// ContainsTag 判断场景是否包含指定标签
func (c *Config) ContainsTag(tag string) bool {
	return slices.Contains(c.Tags, tag)
}

func (c *Config) Validate() error {
	return validate.Struct(c)
}
