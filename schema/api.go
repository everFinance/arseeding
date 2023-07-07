package schema

import (
	"github.com/shopspring/decimal"
)

const (
	AllowStreamMinItemSize = 5 * 1024 * 1024    // 5 MB
	AllowMaxRespDataSize   = 50 * 1024 * 1024   // 50 MB
	SubmitMaxSize          = 1024 * 1024 * 1024 // 1 GB
)

type RespReceiptEverTx struct {
	RawId     uint64 `json:"rawId"` // everTx rawId
	EverHash  string `json:"everHash"`
	Timestamp int64  `json:"timestamp"` // ms
	Symbol    string `json:"symbol"`
	Amount    string `json:"amount"`
	Decimals  int    `json:"decimals"`
}

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

type RespGetOrder struct {
	ID uint `json:"id"`
	RespOrder
	PaymentStatus string `json:"paymentStatus"` // "unpaid", "paid", "expired"
	PaymentId     string `json:"paymentId"`
	OnChainStatus string `json:"onChainStatus"` // "waiting","pending","success","failed"
	Sort          bool   `json:"sort"`
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

type RespApiKey struct {
	EstimateCap string            `json:"estimateCap"`
	Tokens      map[string]TokBal `json:"tokens"` // tokenTag
}

type TokBal struct {
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Balance  string `json:"balance"`
}

type RespErr struct {
	Err string `json:"error"`
}

func (r RespErr) Error() string {
	return r.Err
}
