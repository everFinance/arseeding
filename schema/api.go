package schema

import "math/big"

const (
	AllowMaxItemSize = 100 * 1024 * 1024 // 100 MB
)

type RespOrder struct {
	ItemId             string `json:"ItemId"`   // bundleItem id
	Bundler            string `json:"bundler"`  // fee receiver address
	Currency           string `json:"currency"` // payment token symbol
	Fee                string `json:"fee"`
	PaymentExpiredTime int64  `json:"paymentExpiredTime"`
	ExpectedBlock      int64  `json:"expectedBlock"`
}

type Fee struct {
	Currency string
	Base     *big.Float
	PerChunk *big.Float
}
