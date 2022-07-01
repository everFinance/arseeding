package schema

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

const (
	WaitOnChain    = "waiting"
	PendingOnChain = "pending"
	SuccOnChain    = "success"
	FailedOnChain  = "failed"

	// order payment status
	UnPayment      = "unpaid"
	SuccPayment    = "paid"
	ExpiredPayment = "expired"

	// ReceiptEverTx Status
	UnSpent   = "unspent"
	Spent     = "spent"
	UnRefund  = "unrefund"
	Refund    = "refunded"
	RefundErr = "refundErr"

	MaxPerOnChainSize = 500 * 1024 * 1024 // 500 MB
)

type Order struct {
	gorm.Model
	ItemId   string // bundleItem id
	Signer   string `gorm:"index:idx1"` // item signer
	SignType int

	Size               int64
	Currency           string // payment token symbol
	Decimals           int
	Fee                string
	PaymentExpiredTime int64 // uint s
	ExpectedBlock      int64

	PaymentStatus string // "unpaid", "paid", "expired"
	PaymentId     string // everHash

	OnChainStatus string // "waiting","pending","success","failed"
}

type ReceiptEverTx struct {
	RawId    uint64 `grom:"primarykey"` // everTx rawId
	EverHash string `gorm:"unique"`
	Nonce    int64  // ms
	Symbol   string
	From     string
	Amount   string
	Data     string

	Status string //  "unspent","spent", "unrefund", "refund"
	ErrMsg string
}

type TokenPrice struct {
	Symbol    string `gorm:"primarykey"` // token symbol
	Decimals  int
	Price     float64 // unit is USD
	ManualSet bool    // manual set
	UpdatedAt time.Time
}

type OnChainTx struct {
	gorm.Model
	ArId      string
	CurHeight int64
	Status    string         // "pending","success"
	ItemIds   datatypes.JSON // json.marshal(itemIds)
	ItemNum   int
}
