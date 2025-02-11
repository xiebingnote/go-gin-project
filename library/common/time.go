package common

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"project/library/resource"

	"github.com/jinzhu/now"
)

type TimeType int

// init initializes the now package with a custom week start day.
func init() {
	// Set the week start day to Monday.
	now.WeekStartDay = time.Monday
}

// NowTime returns the current time in Unix timestamp format.
func NowTime() int64 {
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
		// Represents a daytime type
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
		// For the daytime type, the start time is 23 hours ago
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

// GetPgInterval returns the PostgresSQL interval string for a given time type.
//
// This function maps the provided TimeType to a corresponding PostgresSQL interval
// string, which can be used to specify the time range for queries. The mapping
// is as follows:
//   - TimeTypeDay: returns "hour"
//   - TimeTypeWeek: returns "day"
//   - TimeTypeMonth: returns "day"
//   - TimeTypeHour: returns "minute"
//   - TimeType6Hour: returns "hour"
//   - TimeType12Hour: returns "hour"
//
// If the TimeType is not recognized, the function returns an error indicating
// an invalid time type.
func GetPgInterval(timeType TimeType) (string, error) {
	// The PostgresSQL interval is used to specify the time range for a query.
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
	// Return the PostgresSQL interval.
	return interval, nil
}

// GetPgTimeLayout returns the PostgresSQL time layout string for a given time type.
//
// The function maps the provided TimeType to a corresponding PostgresSQL time
// layout string, which can be used to format the time field in a PostgresSQL query.
// The mapping is as follows:
//   - TimeTypeDay: returns "yyyy-MM-dd HH24:00"
//   - TimeTypeWeek: returns "yyyy-MM-dd"
//   - TimeTypeMonth: returns "yyyy-MM-dd"
//   - TimeTypeHour: returns "yyyy-MM-dd HH24:MI"
//   - TimeType6Hour: returns "yyyy-MM-dd HH24:00"
//   - TimeType12Hour: returns "yyyy-MM-dd HH24:00"
//
// If the TimeType is not recognized, the function returns an error indicating
// an invalid time type.
func GetPgTimeLayout(timeType TimeType) (string, error) {
	// The PostgresSQL time layout is used to format the time field in a query.
	// The layout is determined by the time type.
	var layout string
	switch timeType {
	case TimeTypeDay:
		// For TimeTypeDay, the layout is 'yyyy-MM-dd HH24:00'.
		layout = "yyyy-MM-dd HH24:00"
	case TimeTypeWeek:
		fallthrough
	case TimeTypeMonth:
		// For TimeTypeWeek and TimeTypeMonth, the layout is 'yyyy-MM-dd'.
		layout = "yyyy-MM-dd"
	case TimeTypeHour:
		// For TimeTypeHour, the layout is 'yyyy-MM-dd HH24: MI'.
		layout = "yyyy-MM-dd HH24:MI"
	case TimeType6Hour:
		// For TimeType6Hour, the layout is 'yyyy-MM-dd HH24:00'.
		layout = "yyyy-MM-dd HH24:00"
	case TimeType12Hour:
		// For TimeType12Hour, the layout is 'yyyy-MM-dd HH24:00'.
		layout = "yyyy-MM-dd HH24:00"
	default:
		// If the time type is not recognized, return an error.
		return "", fmt.Errorf("invalid time type")
	}
	// Return the PostgresSQL time layout.
	return layout, nil
}

// GetGoTimeLayout returns the Go time layout string for a given time type.
//
// The function maps the provided TimeType to a corresponding Go time
// layout string, which can be used to format the time field in a Go program.
// The mapping is as follows:
//   - TimeTypeDay: returns "2006-01-02 15:00"
//   - TimeTypeWeek: returns "2006-01-02"
//   - TimeTypeMonth: returns "2006-01-02"
//   - TimeTypeHour: returns "2006-01-02 15:04"
//   - TimeType6Hour: returns "2006-01-02 15:00"
//   - TimeType12Hour: returns "2006-01-02 15:00"
//
// If the TimeType is not recognized, the function returns an error indicating
// an invalid time type.
func GetGoTimeLayout(timeType TimeType) (string, error) {
	// The Go time layout is used to format the time field in a Go program.
	// The layout is determined by the time type.
	var layout string
	switch timeType {
	case TimeTypeDay:
		// For TimeTypeDay, the layout is '2006-01-02 15:00'.
		layout = "2006-01-02 15:00"
	case TimeTypeWeek:
		// For TimeTypeWeek, the layout is '2006-01-02'.
		layout = "2006-01-02"
	case TimeTypeMonth:
		// For TimeTypeMonth, the layout is '2006-01-02'.
		layout = "2006-01-02"
	case TimeTypeHour:
		// For TimeTypeHour, the layout is '2006-01-02 15:04'.
		layout = "2006-01-02 15:04"
	case TimeType6Hour:
		// For TimeType6Hour, the layout is '2006-01-02 15:00'.
		layout = "2006-01-02 15:00"
	case TimeType12Hour:
		// For TimeType12Hour, the layout is '2006-01-02 15:00'.
		layout = "2006-01-02 15:00"
	default:
		// If the time type is not recognized, return an error.
		return "", fmt.Errorf("invalid time type")
	}
	// Return the Go time layout.
	return layout, nil
}

// FindTimeWindow finds the time window in which 't' belongs within 'timeSlice'.
// The 'timeSlice' must be sorted in ascending order.
//
// The function returns the starting time of the window that 't' falls into.
// If 't' is before the first window, it returns an error.
// If 'timeSlice' is empty, it returns an error.
func FindTimeWindow(timeSlice []time.Time, t time.Time) (time.Time, error) {
	if len(timeSlice) == 0 {
		// Return an error if the provided timeSlice is empty.
		return time.Time{}, errors.New("empty timeSlice")
	}

	// Iterate through the timeSlice to find the first time greater than 't'.
	for idx := range timeSlice {
		if timeSlice[idx].After(t) {
			// If found, return the previous time as the start of the window.
			if idx > 0 {
				return timeSlice[idx-1], nil
			}
			// If idx is 0, 't' is before the first time window.
			return time.Time{}, errors.New("given time is before the first time window")
		}
	}

	// If 't' is after all time windows, return the last time window.
	return timeSlice[len(timeSlice)-1], nil
}

// SplitTimeRangeByMonth splits the given time range into separate months,
// returning the timestamps for the start of each month, inclusive of the end
// time.
//
// The function takes two int64 arguments, start and end, which are the start
// and end timestamps of the time range to be split. The function returns a
// slice of int64 values, which are the timestamps for the start of each month
// in the given time range.
func SplitTimeRangeByMonth(start, end int64) []int64 {
	s := now.New(Ts2Time(start)).BeginningOfMonth()
	e := now.New(Ts2Time(end)).BeginningOfMonth()

	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.AddDate(0, 1, 0)
	})
}

// SplitTimeRangeByWeek splits the given start and end timestamps into natural weeks,
// returning the timestamps for the start of each week, inclusive of the end time.
//
// The function takes two int64 arguments, start and end, representing the start
// and end timestamps to be split. It returns a slice of int64 values which are
// the timestamps for the start of each week in the given time range.
func SplitTimeRangeByWeek(start, end int64) []int64 {
	// Convert start timestamp to the beginning of its week.
	s := now.New(Ts2Time(start)).BeginningOfWeek()
	// Convert end timestamp to the beginning of its week.
	e := now.New(Ts2Time(end)).BeginningOfWeek()

	// Use the increment function to collect timestamps for the start of each week
	// by adding 7 days to the current week start time until the end time.
	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.AddDate(0, 0, 7)
	})
}

// SplitTimeRangeByDay splits the given start and end timestamps into natural days,
// returning the timestamps for the start of each day, inclusive of the end time.
//
// The function takes two int64 arguments, start and end, which represent the
// start and end timestamps to be split. It returns a slice of int64 values
// which are the timestamps for the start of each day in the given time range.
func SplitTimeRangeByDay(start, end int64) []int64 {
	// Convert start timestamp to the beginning of its day.
	s := now.New(Ts2Time(start)).BeginningOfDay()
	// Convert end timestamp to the beginning of its day.
	e := now.New(Ts2Time(end)).BeginningOfDay()

	// Use the increment function to collect timestamps for the start of each day
	// by adding 1 day to the current day start time until the end time.
	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.AddDate(0, 0, 1)
	})
}

// SplitTimeRangeByHour splits the given start and end timestamps into natural hours,
// returning the timestamps for the start of each hour, inclusive of the end time.
//
// The function takes two int64 arguments, start and end, which represent the
// start and end timestamps to be split. It returns a slice of int64 values
// which are the timestamps for the start of each hour in the given time range.
func SplitTimeRangeByHour(start, end int64) []int64 {
	// Convert start timestamp to the beginning of its hour.
	s := now.New(Ts2Time(start)).BeginningOfHour()
	// Convert end timestamp to the beginning of its hour.
	e := now.New(Ts2Time(end)).BeginningOfHour()

	// Use the increment function to collect timestamps for the start of each hour
	// by adding 1 hour to the current hour start time until the end time.
	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.Add(time.Hour)
	})
}

// SplitTimeRangeBy5Minute splits the given start and end timestamps into natural 5-minute
// intervals, returning the timestamps for the start of each 5-minute interval,
// inclusive of the end time.
//
// The function takes two int64 arguments, start and end, which represent the
// start and end timestamps to be split. It returns a slice of int64 values
// which are the timestamps for the start of each 5-minute interval in the given
// time range.
func SplitTimeRangeBy5Minute(start, end int64) []int64 {
	// Convert start timestamp to the beginning of its 5-minute interval.
	// If the start time is not a 5-minute interval, round it down to the
	// previous 5-minute interval.
	s := Ts2Time(start).Round(time.Minute * 5)
	if s.Unix() > start {
		s = s.Add(-5 * time.Minute)
	}

	// Convert end timestamp to the beginning of its 5-minute interval.
	// If the end time is not a 5-minute interval, round it down to the previous
	// 5-minute interval.
	e := now.New(Ts2Time(end)).BeginningOfMinute()

	// Use the increment function to collect timestamps for the start of each
	// 5-minute interval by adding 5 minutes to the current 5-minute interval
	// start time until the end time.
	return increment(s, e, end, func(time2 time.Time) time.Time {
		return time2.Add(time.Minute * 5)
	})
}

// Increment generates a slice of Unix timestamps by incrementing the start time
// using the provided addFunc until it surpasses or equals the end time.
//
// Parameters:
// - s: the start time.
// - e: the end time (exclusive).
// - end: the Unix timestamp for the inclusive end time.
// - addFunc: a function that takes a time and returns the next increment.
//
// Returns a slice of int64 values representing Unix timestamps.
func increment(s, e time.Time, end int64, addFunc func(time2 time.Time) time.Time) []int64 {
	var result []int64

	// Loop through the time range, appending each Unix timestamp to the result.
	for s.Before(e) || s.Equal(e) {
		result = append(result, s.Unix())
		s = addFunc(s) // Increment the time using the provided function.
	}

	// Ensure the last timestamp is included if it's less than the 'end' timestamp.
	if len(result) > 0 && result[len(result)-1] < end {
		result = append(result, end)
	}

	return result
}

// CreateTimePoint generates a slice of time.Time objects, each representing a
// specific point in time between the provided start and end times, with the
// specified interval.
//
// The interval is specified as a string, e.g. "1h", "2m", "3s", etc. The
// resulting slice will contain each point in time between the start and end
// times, with the specified interval.
//
// For example, if the start time is 2020-01-01 00:00:00 and the end time is
// 2020-01-01 01:00:00, and the interval is "30m", the resulting slice will
// contain the following points in time:
//
//	2020-01-01 00:00:00
//	2020-01-01 00:30:00
//	2020-01-01 01:00:00
func CreateTimePoint(startTime time.Time, endTime time.Time, interval string) (timePoint []time.Time) {
	// Calculate the time interval in milliseconds.
	timeInterval, _ := time.ParseDuration(interval)

	// Truncate the start time to the nearest minute.
	startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), startTime.Hour(), 0, 0, 0, startTime.Location())

	// Calculate the end time by adding the time interval to the start time.
	endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), endTime.Hour(), 0, 0, 0, startTime.Location()).Add(timeInterval)

	// If the interval is 24 hours, truncate the start time to the nearest day.
	if interval == "24h" {
		startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, startTime.Location()).Add(timeInterval)
	}

	// Calculate the number of milliseconds in each interval.
	eachInterval := timeInterval.Milliseconds()

	// Calculate the start time in milliseconds.
	st := startTime
	startTimeMill := st.UnixMilli()

	// Calculate the end time in milliseconds.
	endTimeMill := endTime.UnixMilli()

	// Loop through the range of times, adding each point in time to the
	// resulting slice.
	for startTimeMill < endTimeMill {
		timePoint = append(timePoint, st)
		startTimeMill += eachInterval
		st = st.Add(timeInterval)
	}

	return
}

// GetYearMonthsTimes retrieves the Unix timestamps for the start of each month of the specified year.
// yearStr: a string representing the year (e.g., "2020").
func GetYearMonthsTimes(yearStr string) []int64 {
	// Get the current Unix timestamp.
	n := NowTime()

	// Convert the year string to an integer.
	year := Atoi(yearStr)
	if year == 0 {
		// Log an error if the year could not be parsed.
		resource.LoggerService.Error(fmt.Sprintf("Failed to parse year: %v", yearStr))
		return nil
	}

	// Slice to store the start timestamps of each month.
	var monthStartTimes []int64

	// Iterate over each month (1 to 12).
	for month := 1; month <= 12; month++ {
		// Construct the time for the first day of each month.
		startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		ts := startOfMonth.Unix()

		// Stop if the timestamp exceeds the current time.
		if ts > n {
			break
		}

		// Append the Unix timestamp to the slice.
		monthStartTimes = append(monthStartTimes, ts)
	}

	return monthStartTimes
}

// GetMonthDaysTimes retrieves the Unix timestamps for each day of the specified month.
// monthStr: a string representing the month (e.g., "2020-01").
func GetMonthDaysTimes(monthStr string) []int64 {
	// Parse the input string into a time object.
	timeObj, err := time.Parse("2006-01", monthStr)
	if err != nil {
		// Log an error if the input string could not be parsed.
		resource.LoggerService.Error(fmt.Sprintf("Failed to parse month: %v", err))
		return nil
	}

	// Slice to store the start timestamps of each day.
	var daysTimes []int64

	// Get the start of the month.
	startOfMonth := timeObj
	// Get the start of the next month.
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	// Calculate the number of days in the month.
	daysInMonth := int(startOfNextMonth.Sub(startOfMonth).Hours() / 24)

	// Get the current Unix timestamp.
	n := NowTime()

	// Loop through the range of days, adding each start timestamp to the resulting slice.
	for day := 0; day <= daysInMonth; day++ {
		dayTime := startOfMonth.AddDate(0, 0, day)
		ts := dayTime.Unix()

		// Stop if the timestamp exceeds the current time.
		if ts > n {
			break
		}

		// Append the start timestamp to the slice.
		daysTimes = append(daysTimes, dayTime.Unix())
	}

	return daysTimes
}

// GetWeekDaysTimes 获取指定周的每天的时间戳
// GetWeekDaysTimes gets the Unix timestamps for each day of the specified week.
// weekStr: a string representing the week (e.g., "2020-01-01").
// The input string can be any date of the week, and the function will get the timestamps for the whole week.
func GetWeekDaysTimes(weekStr string) []int64 {
	// Parse the input string into a time object.
	timeObj, err := time.Parse("2006-01-02", weekStr)
	if err != nil {
		// Log an error if the input string could not be parsed.
		resource.LoggerService.Error(fmt.Sprintf("Failed to parse week: %v", err))
		return nil
	}

	week := now.New(timeObj)
	// Get the week object.
	var weekTimes []int64

	// Slice to store the start timestamps of each day.

	// Get the start of the week.
	startOfWeek := week.BeginningOfWeek()

	n := NowTime()
	// Get the current Unix timestamp.

	// Loop through the range of days, adding each start timestamp to the resulting slice.
	for day := 0; day <= 7; day++ {
		dayTime := startOfWeek.AddDate(0, 0, day)
		ts := dayTime.Unix()
		if ts > n {

			// Stop if the timestamp exceeds the current time.
			break
		}
		weekTimes = append(weekTimes, dayTime.Unix())

		// Append the start timestamp to the slice.
	}

	return weekTimes
}

// GetDayHoursTimes gets the Unix timestamps for each hour of the specified day.
// dayStr is a string representing the day (e.g., "2020-01-01").
// The input string can be any date of the day, and the function will get the timestamps for the whole day.
func GetDayHoursTimes(dayStr string) []int64 {
	// Parse the input string into a time object.
	timeObj, err := time.ParseInLocation("2006-01-02", dayStr, time.Local)
	if err != nil {
		// Log an error if the input string could not be parsed.
		resource.LoggerService.Error(fmt.Sprintf("Failed to parse day: %v", err))
		return nil
	}

	var hoursTimes []int64
	// Get the start of the day.
	startOfDay := timeObj
	n := NowTime()
	// Loop through the range of hours, adding each start timestamp to the resulting slice.
	for hour := 0; hour <= 24; hour++ {
		dayTime := startOfDay.Add(time.Hour * time.Duration(hour))
		ts := dayTime.Unix()
		if ts > n {
			// Stop if the timestamp exceeds the current time.
			// hoursTimes = append(hoursTimes, n)
			break
		}
		hoursTimes = append(hoursTimes, dayTime.Unix())
	}

	return hoursTimes
}

// Atoi converts a string to an int.
// If the string is empty, it returns 0.
func Atoi(a string) int {
	// If the string is empty, return 0.
	if a == "" {
		return 0
	}

	// Attempt to parse the string into an int.
	r, e := strconv.Atoi(a)
	// If the parsing is successful, return the int.
	if e == nil {
		return r
	}

	// If the parsing fails, return 0.
	return 0
}

// GetBeginningOfDay gets the start of the day for the given time object.
// It returns a new time object representing the start of the day.
func GetBeginningOfDay(dayTime time.Time) time.Time {
	// Get the year, month, and day from the given time object.
	y, m, d := dayTime.Date()

	// Create a new time object representing the start of the day.
	// The time is set to midnight (00:00:00).
	return time.Date(y, m, d, 0, 0, 0, 0, time.Now().Location())
}

// GetTimeRangeSlice returns a slice of time objects and the interval between them.
// If the number of days is 5 or more, the interval type is set to IntervalTypeDay, and the slice is populated with the start of each day.
// If the number of days is less than 5, the interval type is set to IntervalTypeHour, and the slice is populated with the start of each hour.
func GetTimeRangeSlice(start time.Time, days int) (timeSlice []time.Time, interval int, intervalType string) {
	var isTrue bool
	intervalType = IntervalTypeHour
	if days >= 5 {
		intervalType = IntervalTypeDay
		isTrue = true
	}

	switch isTrue {
	case true: // 5 days or more
		// Populate the slice with the start of each day.
		timeSlice = make([]time.Time, days)
		for i := 0; i < days; i++ {
			timeSlice[i] = start.AddDate(0, 0, i)
		}
		interval = 1
	default: // Less than 5 days
		// Populate the slice with the start of each hour.
		timeSlice = make([]time.Time, 24)
		for i := 0; i < 24; i++ {
			timeSlice[i] = start.Add(time.Duration(i) * time.Hour * time.Duration(days))
		}
		interval = days
	}

	// Sort the slice in ascending order by time.
	sort.Slice(timeSlice, func(i, j int) bool {
		return timeSlice[i].Before(timeSlice[j])
	})

	return timeSlice, interval, intervalType
}

// GetTimeSliceToString converts a slice of time objects into a slice of strings.
// The strings are formatted according to the interval type.
// If the interval type is IntervalTypeHour, the strings are formatted as "2006-01-02 15:04:05".
// Otherwise, the strings are formatted as "2006-01-02 15:04:05".
func GetTimeSliceToString(timeSlice []time.Time, intervalType string) (res []string) {
	// Loop through the slice of time objects.
	for _, v := range timeSlice {
		// Use a switch statement to format the strings according to the interval type.
		switch intervalType {
		case IntervalTypeHour:
			// If the interval type is IntervalTypeHour, format the strings as "2006-01-02 15:04:05".
			date := time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), 0, 0, 0, time.Local)
			res = append(res, date.Format("2006-01-02 15:04:05"))
		default:
			// Otherwise, format the strings as "2006-01-02 15:04:05".
			date := time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.Local)
			res = append(res, date.Format("2006-01-02 15:04:05"))
		}
	}
	return res
}

// GetDurationDays calculates the number of days represented by the given duration.
// The duration is calculated as follows:
//   - The hours of the duration are divided by 24 to get the number of days.
//   - If the remainder is not zero, another day is added.
func GetDurationDays(duration time.Duration) int {
	// Calculate the number of days represented by the given duration.
	days := int(duration.Hours() / 24)
	// If the remainder is not zero, add one more day.
	if int(duration.Hours())%24 != 0 {
		days++
	}
	// Return the calculated number of days.
	return days
}

// FormatTimeSlice formats a slice of time.Time objects into a slice of strings.
// The format of the strings is determined by the given time interval type.
// If the time interval type is IntervalTypeHour, the format is "2006-01-02 15:04".
// Otherwise, the format is "2006-01-02".
func FormatTimeSlice(timeIntervalType string, timeSlice []time.Time) []string {
	// Initialize a slice to store the formatted time strings.
	var res = make([]string, 0)

	// Iterate over each time in the slice.
	for _, v := range timeSlice {
		if timeIntervalType == IntervalTypeHour {
			// Format the time as "2006-01-02 15:04" if the interval type is IntervalTypeHour.
			res = append(res, v.Format("2006-01-02 15:04"))
		} else {
			// Otherwise, format the time as "2006-01-02".
			res = append(res, v.Format("2006-01-02"))
		}
	}

	// Return the slice of formatted time strings.
	return res
}
