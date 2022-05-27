package schema

import (
	"github.com/shopspring/decimal"
)

const (
	AllowMaxItemSize = 100 * 1024 * 1024 // 100 MB
)

type RespOrder struct {
	ItemId             string `json:"ItemId"`   // bundleItem id
	Bundler            string `json:"bundler"`  // fee receiver address
	Currency           string `json:"currency"` // payment token symbol
	Decimals           int    `json:"decimals"`
	Fee                string `json:"fee"`
	PaymentExpiredTime int64  `json:"paymentExpiredTime"`
	ExpectedBlock      int64  `json:"expectedBlock"`
}

type Fee struct {
	Currency string          `json:"currency"`
	Decimals int             `json:"decimals"`
	Base     decimal.Decimal `json:"base"`
	PerChunk decimal.Decimal `json:"perChunk"`
}

type RespFee struct {
	Currency string `json:"currency"`
	Decimals int    `json:"decimals"`
	FinalFee string `json:"finalFee"`
}
