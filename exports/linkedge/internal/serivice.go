package internal

import "github.com/ibuilding-x/driver-box/exports/linkedge/model"

type Service interface {
	// Create 创建新的场景联动配置
	// 参数: config - 场景联动配置对象
	// 返回: error - 错误信息
	Create(model.Config) error

	// Update 更新已存在的场景联动配置
	// 参数: config - 更新后的场景联动配置对象
	// 返回: error - 错误信息
	Update(model.Config) error

	// Delete 删除指定ID的场景联动配置
	// 参数: id - 场景联动配置的唯一标识符
	// 返回: error - 错误信息
	Delete(id string) error

	// Trigger 手动触发指定ID的场景联动
	// 参数: id - 场景联动配置的唯一标识符
	// 返回: error - 错误信息
	Trigger(id string) error

	// Execute 执行场景联动配置
	// 参数: config - 要执行的场景联动配置对象
	// 返回: error - 错误信息
	Execute(config model.Config) error
}
