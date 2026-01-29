package linkedge

import "github.com/ibuilding-x/driver-box/v2/exports/linkedge/model"

type IService interface {
	Get(id string) (model.Config, error)
	// GetList 获取场景联动列表
	// 参数: tag - 标签列表
	GetList(tag ...string) ([]model.Config, error)
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
