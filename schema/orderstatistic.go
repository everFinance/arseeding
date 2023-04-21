package schema

import (
	"time"
)

type Result struct {
	Status        string
	Totals        int64
	TotalDataSize int64
}

type Pending struct {
	Porders   int64
	Pdatasize int64
}

type Success struct {
	Sorders   int64
	Sdatasize int64
}

type Waitting struct {
	Worders   int64
	Wdatasize int64
}

type Failed struct {
	Forders   int64
	Fdatasize int64
}

type Range struct {
	Start string
	End   string
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
