package linkage

import "time"

// intInSlice 检查数字是否在切片中
func intInSlice(num int, nums []int) bool {
	for i, _ := range nums {
		if num == nums[i] {
			return true
		}
	}
	return false
}

// Verify 验证条件是否满足
func (c YearsCondition) Verify(year int) bool {
	return intInSlice(year, c.Years)
}

// Verify 验证条件是否满足
func (c MonthsCondition) Verify(month int) bool {
	return intInSlice(month, c.Months)
}

// Verify 验证条件是否满足
func (c DaysCondition) Verify(day int) bool {
	return intInSlice(day, c.Days)
}

// Verify 验证条件是否满足
func (c WeeksCondition) Verify(week int) bool {
	return intInSlice(week, c.Weeks)
}

// Verify 验证条件是否满足
func (c TimeIntervalCondition) Verify(t time.Time) bool {
	beginTime, _ := time.Parse("15:04", c.BeginTime)
	endTime, _ := time.Parse("15:04", c.EndTime)

	beginTime = beginTime.AddDate(t.Year(), int(t.Month()), t.Day())
	endTime = endTime.AddDate(t.Year(), int(t.Month()), t.Day())

	if t.After(beginTime) && t.Before(endTime) {
		return true
	}
	return false
}

// Check 检查条件是否合规
func (c YearsCondition) Check() bool {
	for i, _ := range c.Years {
		if c.Years[i] < 0 || c.Years[i] > 9999 {
			return false
		}
	}
	return true
}

// Check 检查条件是否合规
func (c MonthsCondition) Check() bool {
	for i, _ := range c.Months {
		if c.Months[i] <= 0 || c.Months[i] > 12 {
			return false
		}
	}
	return true
}

// Check 检查条件是否合规
func (c DaysCondition) Check() bool {
	for i, _ := range c.Days {
		if c.Days[i] <= 0 || c.Days[i] > 31 {
			return false
		}
	}
	return true
}

// Check 检查条件是否合规
func (c WeeksCondition) Check() bool {
	for i, _ := range c.Weeks {
		if c.Weeks[i] < 0 || c.Weeks[i] > 6 {
			return false
		}
	}
	return true
}

// Check 检查条件是否合规
func (c TimeIntervalCondition) Check() bool {
	beginTime, _ := time.Parse("15:04", c.BeginTime)
	endTime, _ := time.Parse("15:04", c.EndTime)

	if beginTime.After(endTime) {
		return false
	}
	return true
}
