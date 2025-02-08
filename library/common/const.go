package common

import "time"

const (
	DefaultTimeLayout    = time.RFC3339
	LayoutDateTime       = "2006-01-02 15:04:05"
	LayoutDate           = "2006-01-02"
	LayoutMonth          = "2006-01"
	LayoutDateHourNoDash = "2006010215"
)

const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
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
