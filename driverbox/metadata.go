package driverbox

import (
	"github.com/ibuilding-x/driver-box/internal/core"
	"github.com/ibuilding-x/driver-box/pkg/config"
)

//// 获取当前被注册至 driver-box 的所有export
//func GetExports() []export.Export {
//	return export0.Exports
//}

// UpdateMetadata 更新服务元数据信息
// 参数:
//   - f: 用于修改元数据的函数，接收元数据指针作为参数
//
// 使用示例:
//
//	driverbox.UpdateMetadata(func(metadata *config.Metadata) {
//	    metadata.SerialNo = "new_serial_no"
//	})
func UpdateMetadata(f func(*config.Metadata)) {
	f(&core.Metadata)
}

// GetMetadata 获取服务元数据信息
// 返回当前服务的核心元数据配置，包括序列号等关键信息
// 返回值:
//   - config.Metadata: 当前服务的元数据配置
//
// 使用示例:
//
//	metadata := driverbox.GetMetadata()
//	driverbox.Log().Info("Service Serial Number", zap.String("serial", metadata.SerialNo))
func GetMetadata() config.Metadata {
	return core.Metadata
}
