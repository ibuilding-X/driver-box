package driverbox

import "github.com/ibuilding-x/driver-box/pkg/crontab"

// AddFunc 添加定时任务
// 参数:
//   - s: cron表达式，定义任务执行的时间计划
//   - f: 需要执行的任务函数
//
// 返回值:
//   - *crontab.Future: 定时任务的Future对象，可用于取消任务
//   - error: 操作过程中发生的错误
//
// 支持的cron格式示例:
//   - "* * * * *" 每分钟执行
//   - "0 0 * * *" 每天零点执行
func AddFunc(s string, f func()) (*crontab.Future, error) {
	return crontab.Instance().AddFunc(s, f)
}
