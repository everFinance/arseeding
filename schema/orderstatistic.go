package schema

import (
	"time"
)

type Result struct {
	Status        string `json:"status"`
	Totals        int64  `json:"totals"`
	TotalDataSize int64  `json:"totalDataSize"`
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
	Date   string `json:"date"`
	Result Result `json:"result"`
}

type OrderStatistic struct {
	ID            uint      `gorm:"primarykey"`
	Date          time.Time `json:"date"`
	Totals        int64     `json:"totals"`
	TotalDataSize int64     `json:"totalDataSize"`
}
