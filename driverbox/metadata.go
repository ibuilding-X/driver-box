package driverbox

import (
	"github.com/ibuilding-x/driver-box/v2/internal/core"
	"github.com/ibuilding-x/driver-box/v2/pkg/config"
)

//// 获取当前被注册至 driver-box 的所有export
//func GetExports() []export.Export {
//	return export0.Exports
//}

// UpdateMetadata 更新服务元数据信息
// 允许动态修改服务的元数据配置，如序列号、版本等信息
// 参数:
//   - f: 用于修改元数据的函数，接收元数据指针作为参数
//     通过此函数可以安全地修改元数据字段
//
// 使用示例:
//
//	driverbox.UpdateMetadata(func(metadata *config.Metadata) {
//	    metadata.SerialNo = "new_serial_no"
//	    metadata.Version = "2.0.0"
//	})
func UpdateMetadata(f func(*config.Metadata)) {
	f(&core.Metadata)
}

// GetMetadata 获取服务元数据信息
// 返回当前服务的核心元数据配置，包括序列号、版本等关键信息
// 返回值:
//   - config.Metadata: 当前服务的元数据配置副本
//
// 使用示例:
//
//	metadata := driverbox.GetMetadata()
//	driverbox.Log().Info("Service Serial Number", zap.String("serial", metadata.SerialNo))
//	driverbox.Log().Info("Service Version", zap.String("version", metadata.Version))
func GetMetadata() config.Metadata {
	return core.Metadata
}
