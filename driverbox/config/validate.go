package config

import "github.com/go-playground/validator/v10"

// Validate 核心配置文件校验
func (c Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}
