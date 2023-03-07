package schema

import (
	"github.com/shopspring/decimal"
)

const (
	AllowMaxItemSize       = 1 * 1024 * 1024   // 1 MB
	AllowMaxNativeDataSize = 1 * 1024 * 1024   // 1 MB
	AllowMaxRespDataSize   = 500 * 1024 * 1024 // 500 MB
)

type RespOrder struct {
	ItemId             string `json:"itemId"` // bundleItem id
	Size               int64  `json:"size"`
	Bundler            string `json:"bundler"`  // fee receiver address
	Currency           string `json:"currency"` // payment token symbol
	Decimals           int    `json:"decimals"`
	Fee                string `json:"fee"`
	PaymentExpiredTime int64  `json:"paymentExpiredTime"`
	ExpectedBlock      int64  `json:"expectedBlock"`
}

type RespItemId struct {
	ItemId string `json:"itemId"` // bundleItem id
	Size   int64  `json:"size"`
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
	FinalFee string `json:"finalFee"` // uint
}

type ResBundler struct {
	Bundler string `json:"bundler"`
}

type RespErr struct {
	Err string `json:"error"`
}

func (r RespErr) Error() string {
	return r.Err
}
