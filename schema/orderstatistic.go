package schema

import (
	"time"
)

type Result struct {
	Status        string
	Totals        int64
	TotalDataSize int64
}

type Range struct {
	Start string
	End   string
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}
type DailyStatistic struct {
	Date    string
	Results []*Result
}

type OrderStatistic struct {
	Date          time.Time `gorm:"primary_key"`
	Status        string    `gorm:"primary_key"`
	Totals        int64
	TotalDataSize int64
}
