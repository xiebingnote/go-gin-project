package common

import (
	"errors"
	"fmt"
	"project/library/resource"
	"sort"
	"strconv"
	"time"

	"github.com/jinzhu/now"
)

const (
	LayoutDateTime       = "2006-01-02 15:04:05"
	LayoutDate           = "2006-01-02"
	LayoutMonth          = "2006-01"
	LayoutDateHourNoDash = "2006010215"
)

const (
	TimeTypeDay TimeType = iota + 1
	TimeTypeWeek
	TimeTypeMonth
	TimeTypeHour
	TimeType6Hour
	TimeType12Hour
)
const (
	IntervalTypeHour = "hour"
	IntervalTypeDay  = "day"
)

type TimeType int

// init initializes the now package with a custom week start day.
func init() {
	// Set the week start day to Monday.
	now.WeekStartDay = time.Monday
}

// Ntime returns the current time in Unix timestamp format.
func Ntime() int64 {
	return time.Now().Unix()
}

// Ts2Time converts a Unix timestamp in seconds to a time.Time object.
func Ts2Time(ts int64) time.Time {
	return time.Unix(ts, 0)
}

// Ns2Time converts a Unix nanosecond timestamp to a time.Time object.
func Ns2Time(ns int64) time.Time {
	return time.Unix(0, ns)
}

// GetLastMonthTimeRange returns the time range for the last month.
//
// The time range is returned as a tuple of two time.Time objects, where the
// first element is the start of the last month, and the second element is the
// end of the last month.
//
// The time range is inclusive, meaning that the start and end times are
// included in the range.
func GetLastMonthTimeRange() (time.Time, time.Time) {
	return now.BeginningOfMonth().AddDate(0, -1, 0), now.EndOfMonth().AddDate(0, -1, 0)
}

// GetLastWeekTimeRange returns the time range for the last week.
//
// The time range is returned as a tuple of two time.Time objects, where the
// first element is the start of the last week, and the second element is the
// end of the last week.
//
// The time range is inclusive, meaning that the start and end times are
// included in the range.
func GetLastWeekTimeRange() (time.Time, time.Time) {
	return now.BeginningOfWeek().AddDate(0, 0, -7), now.EndOfWeek().AddDate(0, 0, -7)
}

// GetYesterdayTimeRange returns the time range for yesterday.
//
// The time range is returned as a tuple of two time.Time objects, where the
// first element is the start of yesterday, and the second element is the
// end of yesterday.
//
// The time range is inclusive, meaning that the start and end times are
// included in the range.
func GetYesterdayTimeRange() (time.Time, time.Time) {
	return now.BeginningOfDay().AddDate(0, 0, -1), now.EndOfDay().AddDate(0, 0, -1)
}

// GetBeforeYesterdayTimeRange returns the time range for the day before yesterday.
//
// The time range is returned as a tuple of two time.Time objects, where the
// first element is the start of the day before yesterday, and the second
// element is the end of the day before yesterday.
//
// The time range is inclusive, meaning that the start and end times are
// included in the range.
func GetBeforeYesterdayTimeRange() (time.Time, time.Time) {
	return now.BeginningOfDay().AddDate(0, 0, -2), now.EndOfDay().AddDate(0, 0, -2)
}

// GetBeforeLastWeekTimeRange returns the time range for the week before last week.
//
// The time range is returned as a tuple of two time.Time objects, where the
// first element is the start of the week before last week, and the second
// element is the end of the week before last week.
//
// The time range is inclusive, meaning that the start and end times are
// included in the range.
func GetBeforeLastWeekTimeRange() (time.Time, time.Time) {
	return now.BeginningOfWeek().AddDate(0, 0, -14), now.EndOfWeek().AddDate(0, 0, -14)
}

// GetBeforeLastMonthTimeRange returns the time range for the month before last month.
//
// The time range is returned as a tuple of two time.Time objects, where the
// first element is the start of the month before last month, and the second
// element is the end of the month before last month.
//
// The time range is inclusive, meaning that the start and end times are
// included in the range.
func GetBeforeLastMonthTimeRange() (time.Time, time.Time) {
	return now.BeginningOfMonth().AddDate(0, -2, 0), now.EndOfMonth().AddDate(0, -2, 0)
}

// String returns the string representation of the TimeType.
//
// This method converts the TimeType enumeration value into a human-readable
// string format. If the TimeType is not recognized, it returns an empty string.
func (t TimeType) String() string {
	switch t {
	case TimeTypeDay:
		// Represents a day time type
		return "day"
	case TimeTypeWeek:
		// Represents a week time type
		return "week"
	case TimeTypeMonth:
		// Represents a month time type
		return "month"
	case TimeTypeHour:
		// Represents an hour time type
		return "hour"
	case TimeType6Hour:
		// Represents a 6-hour time type
		return "6hour"
	case TimeType12Hour:
		// Represents a 12-hour time type
		return "12hour"
	default:
		// Default case for unrecognized time types
		return ""
	}
}

// GetTimeByTimeType returns the start and end times for the given time type.
//
// The start time is the beginning of the time range for the given time type,
// and the end time is the end of the time range for the given time type. If
// the time type is not recognized, it returns an error.
func GetTimeByTimeType(timeType TimeType) (time.Time, time.Time, error) {
	var start, end time.Time
	switch timeType {
	case TimeTypeDay:
		// For the day time type, the start time is 23 hours ago
		start = time.Now().Add(-23 * time.Hour)
	case TimeTypeWeek:
		// For the week time type, the start time is 6 days ago
		start = time.Now().AddDate(0, 0, -6)
	case TimeTypeMonth:
		// For the month time type, the start time is 29 days ago
		start = time.Now().AddDate(0, 0, -29)
	case TimeTypeHour:
		// For the hour time type, the start time is 50 minutes ago
		start = time.Now().Add(-50 * time.Minute)
	case TimeType6Hour:
		// For the 6-hour time type, the start time is 5 hours ago
		start = time.Now().Add(-5 * time.Hour)
	case TimeType12Hour:
		// For the 12-hour time type, the start time is 11 hours ago
		start = time.Now().Add(-11 * time.Hour)
	default:
		// Default case for unrecognized time types
		return time.Time{}, time.Time{}, errors.New("invalid time type")
	}
	// The end time is always the end of the day
	end = now.EndOfDay()
	return start, end, nil
}

// GetTimeRangeSliceByTimeType returns a slice of time.Time objects for the given time type.
//
// The slice of time.Time objects represents the time range for the given time type.
// The time range is inclusive, meaning that the start and end times are included in the range.
// The time range is sorted in ascending order.
func GetTimeRangeSliceByTimeType(timeType TimeType) ([]time.Time, error) {
	var timeSlice []time.Time

	switch timeType {
	case TimeTypeDay: // 过去一天，每小时一个
		// Create a slice of 24 time.Time objects, each representing a different hour of the past day.
		timeSlice = make([]time.Time, 24)
		nowHour := now.BeginningOfHour()
		for i := 0; i < 24; i++ {
			// For each hour, subtract the current hour from the current time to get the time for that hour.
			timeSlice[i] = nowHour.Add(-time.Duration(i) * time.Hour)
		}
	case TimeTypeWeek: // 过去一周，每天
		// Create a slice of 7 time.Time objects, each representing a different day of the past week.
		timeSlice = make([]time.Time, 7)
		nowDay := now.BeginningOfDay()
		for i := 0; i < 7; i++ {
			// For each day, subtract the current day from the current time to get the time for that day.
			timeSlice[i] = nowDay.AddDate(0, 0, -i)
		}
	case TimeTypeMonth: // 过去一个月，每天
		// Create a slice of 30 time.Time objects, each representing a different day of the past month.
		timeSlice = make([]time.Time, 30)
		nowDay := now.BeginningOfDay()
		for i := 0; i < 30; i++ {
			// For each day, subtract the current day from the current time to get the time for that day.
			timeSlice[i] = nowDay.AddDate(0, 0, -i)
		}
	case TimeTypeHour: // 过去一小时，每10分钟
		// Create a slice of 6 time.Time objects, each representing a different minute of the past hour.
		timeSlice = make([]time.Time, 6)
		nowMin := now.BeginningOfMinute()
		for i := 0; i < 6; i++ {
			// For each minute, subtract the current minute from the current time to get the time for that minute.
			timeSlice[i] = nowMin.Add(-time.Duration(i*10) * time.Minute)
		}
	case TimeType6Hour: // 过去6小时，每小时
		// Create a slice of 6 time.Time objects, each representing a different hour of the past 6 hours.
		timeSlice = make([]time.Time, 6)
		nowHour := now.BeginningOfHour()
		for i := 0; i < 6; i++ {
			// For each hour, subtract the current hour from the current time to get the time for that hour.
			timeSlice[i] = nowHour.Add(-time.Duration(i) * time.Hour)
		}
	case TimeType12Hour: // 过去12小时，每小时
		// Create a slice of 12 time.Time objects, each representing a different hour of the past 12 hours.
		timeSlice = make([]time.Time, 12)
		nowHour := now.BeginningOfHour()
		for i := 0; i < 12; i++ {
			// For each hour, subtract the current hour from the current time to get the time for that hour.
			timeSlice[i] = nowHour.Add(-time.Duration(i) * time.Hour)
		}
	default:
		// Default case for unrecognized time types
		return nil, errors.New("unsupported time type")
	}

	// Sort the slice of time.Time objects in ascending order.
	sort.Slice(timeSlice, func(i, j int) bool {
		return timeSlice[i].Before(timeSlice[j])
	})

	return timeSlice, nil
}

// GetPgInterval returns the PostgreSQL interval for the given time type.
func GetPgInterval(timeType TimeType) (string, error) {
	// The PostgreSQL interval is used to specify the time range for a query.
	// The interval is determined by the time type.
	var interval string
	switch timeType {
	case TimeTypeDay:
		// For TimeTypeDay, the interval is 'hour'.
		interval = "hour"
	case TimeTypeWeek:
		fallthrough
	case TimeTypeMonth:
		// For TimeTypeWeek and TimeTypeMonth, the interval is 'day'.
		interval = "day"
	case TimeTypeHour:
		// For TimeTypeHour, the interval is 'minute'.
		interval = "minute"
	case TimeType6Hour:
		// For TimeType6Hour, the interval is 'hour'.
		interval = "hour"
	case TimeType12Hour:
		// For TimeType12Hour, the interval is 'hour'.
		interval = "hour"
	default:
		// If the time type is not recognized, return an error.
		return "", fmt.Errorf("invalid time type")
	}
	// Return the PostgreSQL interval.
	return interval, nil
}

// GetPgTimeLayout 获取pg时间格式
func GetPgTimeLayout(timeType TimeType) (string, error) {
	var layout string
	switch timeType {
	case TimeTypeDay:
		layout = "yyyy-MM-dd HH24:00"
	case TimeTypeWeek:
		layout = "yyyy-MM-dd"
	case TimeTypeMonth:
		layout = "yyyy-MM-dd"
	case TimeTypeHour:
		layout = "yyyy-MM-dd HH24:MI"
	case TimeType6Hour:
		layout = "yyyy-MM-dd HH24:00"
	case TimeType12Hour:
		layout = "yyyy-MM-dd HH24:00"
	default:
		return "", fmt.Errorf("invalid time type")
	}
	return layout, nil
}

// GetGoTimeLayout 获取go时间格式
func GetGoTimeLayout(timeType TimeType) (string, error) {
	var layout string
	switch timeType {
	case TimeTypeDay:
		layout = "2006-01-02 15:00"
	case TimeTypeWeek:
		layout = "2006-01-02"
	case TimeTypeMonth:
		layout = "2006-01-02"
	case TimeTypeHour:
		layout = "2006-01-02 15:04"
	case TimeType6Hour:
		layout = "2006-01-02 15:00"
	case TimeType12Hour:
		layout = "2006-01-02 15:00"
	default:
		return "", fmt.Errorf("invalid time type")
	}
	return layout, nil
}

// FindTimeWindow 查找 t 属于 timeSlice 哪个时间窗口,timeSlice需要从小到大排序
func FindTimeWindow(timeSlice []time.Time, t time.Time) (time.Time, error) {
	if len(timeSlice) == 0 {
		return time.Time{}, errors.New("empty timeSlice")
	}

	// 遍历 timeSlice，找到第一个比 t 大的时间
	for idx := range timeSlice {
		if timeSlice[idx].After(t) {
			// 返回该时间窗口的前一个时间
			if idx > 0 {
				return timeSlice[idx-1], nil
			}
			// 如果 idx 为 0，表示 t 在第一个时间窗口之前
			return time.Time{}, errors.New("given time is before the first time window")
			// return timeSlice[0], nil
		}
	}

	// 如果 t 在所有时间窗口之后，返回最后一个时间窗口
	return timeSlice[len(timeSlice)-1], nil
}

// SplitTimeRangeByMonth 将开始和结束时间戳按照自然月分割，返回时间戳，包含结束时间
func SplitTimeRangeByMonth(start, end int64) []int64 {
	s := now.New(Ts2Time(start)).BeginningOfMonth()
	e := now.New(Ts2Time(end)).BeginningOfMonth()

	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.AddDate(0, 1, 0)
	})
}

// SplitTimeRangeByWeek 将开始和结束时间戳按照自然周分割，返回时间戳，包含结束时间
func SplitTimeRangeByWeek(start, end int64) []int64 {
	s := now.New(Ts2Time(start)).BeginningOfWeek()
	e := now.New(Ts2Time(end)).BeginningOfWeek()

	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.AddDate(0, 0, 7)
	})
}

// SplitTimeRangeByDay 将开始和结束时间戳按照自然天分割，返回时间戳，包含结束时间
func SplitTimeRangeByDay(start, end int64) []int64 {
	s := now.New(Ts2Time(start)).BeginningOfDay()
	e := now.New(Ts2Time(end)).BeginningOfDay()

	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.AddDate(0, 0, 1)
	})
}

// SplitTimeRangeByHour 将开始和结束时间戳按照自然小时分割，返回时间戳，包含结束时间
func SplitTimeRangeByHour(start, end int64) []int64 {
	s := now.New(Ts2Time(start)).BeginningOfHour()
	e := now.New(Ts2Time(end)).BeginningOfHour()

	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.Add(time.Hour)
	})
}

// SplitTimeRangeBy5Minute 将开始和结束时间戳按照整5分钟分割，返回时间戳，包含结束时间
func SplitTimeRangeBy5Minute(start, end int64) []int64 {
	s := Ts2Time(start).Round(time.Minute * 5)
	if s.Unix() > start {
		s = s.Add(-5 * time.Minute)
	}

	// 结束时间戳的本月结束时间,0点整
	e := now.New(Ts2Time(end)).BeginningOfMinute()

	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.Add(time.Minute * 5)
	})
}

func increment(s, e time.Time, end int64, addFunc func(time2 time.Time) time.Time) []int64 {
	var result []int64
	for s.Before(e) || s.Equal(e) {
		result = append(result, s.Unix())
		s = addFunc(s)
	}

	if len(result) > 0 && result[len(result)-1] < end {
		result = append(result, end)
	}
	return result
}

func CreateTimePoint(startTime time.Time, endTime time.Time, interval string) (timePoint []time.Time) {
	// create timePoint
	timeInterval, _ := time.ParseDuration(interval)
	startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), startTime.Hour(), 0, 0, 0, startTime.Location())
	endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), endTime.Hour(), 0, 0, 0, startTime.Location()).Add(timeInterval)
	if interval == "24h" {
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, startTime.Location()).Add(timeInterval)
	}
	eachInterval := timeInterval.Milliseconds()
	st := startTime
	startTimeMill := st.UnixMilli()
	endTimeMill := endTime.UnixMilli()
	for startTimeMill < endTimeMill {
		timePoint = append(timePoint, st)
		startTimeMill += eachInterval
		st = st.Add(timeInterval)
	}
	return
}

// GetYearMonthsTimes 获取指定年的每月的时间戳
// yearStr: 2020
func GetYearMonthsTimes(yearStr string) []int64 {
	n := Ntime()
	year := Atoi(yearStr)
	if year == 0 {
		resource.LoggerService.Error(fmt.Sprintf("解析年份失败: %v", yearStr))
		return nil
	}
	var monthStartTimes []int64

	for month := 1; month <= 13; month++ {
		// 构造每个月的第一天的时间
		startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		ts := startOfMonth.Unix()
		if ts > n {
			// 超出添加当前时间
			// monthStartTimes = append(monthStartTimes, n)
			break
		}
		// 转换为Unix时间戳并追加到切片中
		monthStartTimes = append(monthStartTimes, ts)
	}

	return monthStartTimes
}

// GetMonthDaysTimes 获取指定月的每天的时间戳
// monthStr: 2020-01
func GetMonthDaysTimes(monthStr string) []int64 {
	// 解析输入的字符串为时间
	timeObj, err := time.Parse("2006-01", monthStr)
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("解析时间失败: %v", err))
		return nil
	}

	var daysTimes []int64
	// 获取该月的开始时间
	startOfMonth := timeObj
	// 获取下个月的开始时间
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	// 计算该月有多少天
	daysInMonth := startOfNextMonth.Sub(startOfMonth).Hours() / 24

	n := Ntime()
	// 生成每天的开始时间戳
	for day := 0; day <= int(daysInMonth); day++ {
		dayTime := startOfMonth.AddDate(0, 0, day)
		ts := dayTime.Unix()
		if ts > n {
			// 超出添加当前时间
			// monthTimes = append(monthTimes, n)
			break
		}
		daysTimes = append(daysTimes, dayTime.Unix())
	}

	return daysTimes
}

// GetWeekDaysTimes 获取指定周的每天的时间戳
// weekStr: 2020-01-01, 传当周的其中的日期即可
func GetWeekDaysTimes(weekStr string) []int64 {
	// 解析输入的字符串为时间
	timeObj, err := time.Parse("2006-01-02", weekStr)
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("解析时间失败: %v", err))
		return nil
	}

	week := now.New(timeObj)
	var weekTimes []int64
	// 获取该周的开始时间
	startOfWeek := week.BeginningOfWeek()

	n := Ntime()
	// 生成每天的开始时间戳
	for day := 0; day <= 7; day++ {
		dayTime := startOfWeek.AddDate(0, 0, day)
		ts := dayTime.Unix()
		if ts > n {
			// 超出添加当前时间
			// weekTimes = append(weekTimes, n)
			break
		}
		weekTimes = append(weekTimes, dayTime.Unix())
	}

	return weekTimes
}

// GetDayHoursTimes 获取指定天的每个小时的时间戳
// dayStr: 2020-01-01
func GetDayHoursTimes(dayStr string) []int64 {
	// 解析输入的字符串为时间
	timeObj, err := time.ParseInLocation("2006-01-02", dayStr, time.Local)
	if err != nil {
		resource.LoggerService.Error(fmt.Sprintf("解析时间失败: %v", err))
		return nil
	}

	var hoursTimes []int64
	// 获取当天的开始时间
	startOfDay := timeObj
	n := Ntime()
	// 生成每天的开始时间戳
	for hour := 0; hour <= 24; hour++ {
		dayTime := startOfDay.Add(time.Hour * time.Duration(hour))
		ts := dayTime.Unix()
		if ts > n {
			// 超出添加当前时间
			// hoursTimes = append(hoursTimes, n)
			break
		}
		hoursTimes = append(hoursTimes, dayTime.Unix())
	}

	return hoursTimes
}

func Atoi(a string) int {
	if a == "" {
		return 0
	}
	r, e := strconv.Atoi(a)
	if e == nil {
		return r
	}
	return 0
}

func GetBeginningOfDay(dayTime time.Time) time.Time {
	y, m, d := dayTime.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Now().Location())
}

func GetTimeRangeSlice(start time.Time, days int) (timeSlice []time.Time, interval int, intervalType string) {
	var isTrue bool
	intervalType = IntervalTypeHour
	if days >= 5 {
		intervalType = IntervalTypeDay
		isTrue = true
	}

	switch isTrue {
	case true: // 5 天以上
		timeSlice = make([]time.Time, days)
		for i := 0; i < days; i++ {
			timeSlice[i] = start.AddDate(0, 0, i)
		}
		interval = 1
	default: // 5 天以下
		timeSlice = make([]time.Time, 24)
		for i := 0; i < 24; i++ {
			timeSlice[i] = start.Add(time.Duration(i) * time.Hour * time.Duration(days))
		}
		interval = days
	}

	sort.Slice(timeSlice, func(i, j int) bool {
		return timeSlice[i].Before(timeSlice[j])
	})

	return timeSlice, interval, intervalType
}

func GetTimeSliceToString(timeSlice []time.Time, intervalType string) (res []string) {
	for _, v := range timeSlice {
		switch intervalType {
		case IntervalTypeHour:
			date := time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), 0, 0, 0, time.Local)
			res = append(res, date.Format("2006-01-02 15:04:05"))
		default:
			date := time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.Local)
			res = append(res, date.Format("2006-01-02 15:04:05"))
		}
	}
	return res
}

func GetDurationDays(duration time.Duration) int {
	days := int(duration.Hours() / 24)
	if int(duration.Hours())%24 != 0 {
		days++
	}
	return days
}

func FormatTimeSlice(timeIntervalType string, timeSlice []time.Time) []string {
	var res = make([]string, 0)
	for _, v := range timeSlice {
		if timeIntervalType == IntervalTypeHour {
			res = append(res, v.Format("2006-01-02 15:04"))
		} else {
			res = append(res, v.Format("2006-01-02"))
		}
	}
	return res
}
