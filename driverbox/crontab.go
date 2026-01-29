package driverbox

import "github.com/ibuilding-x/driver-box/v2/pkg/crontab"

// AddFunc 添加定时任务
// 参数:
//   - s: cron表达式，定义任务执行的时间计划
//     支持标准cron格式: 秒 分 时 日 月 星期
//     示例: "* * * * *" 每分钟执行, "0 0 * * *" 每天零点执行
//   - f: 需要执行的任务函数，无参数无返回值
//
// 返回值:
//   - *crontab.Future: 定时任务的Future对象，可用于取消任务或查询状态
//   - error: 操作过程中发生的错误，如cron表达式格式错误等
//
// 使用示例:
//
//	future, err := driverbox.AddFunc("0 */5 * * * *", func() {
//	    driverbox.Log().Info("定时任务执行")
//	})
//	if err != nil {
//	    driverbox.Log().Error("添加定时任务失败", zap.Error(err))
//	}
func AddFunc(s string, f func()) (*crontab.Future, error) {
	return crontab.Instance().AddFunc(s, f)
}
