package schema

import (
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
	UnSpent  = "unspent"
	Spent    = "spent"
	UnRefund = "unrefund"
	Refund   = "refund"
)

type Order struct {
	gorm.Model
	ItemId   string // bundleItem id
	Signer   string // item signer
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
	gorm.Model
	EverHash string `gorm:"unique"`
	Nonce    int64  // ms
	Symbol   string
	Action   string
	From     string
	Amount   string
	Data     string
	Page     int `gorm:"index:idx1"`

	Status string //  "unspent","spent", "unrefund", "refund"
}

type TokenPrice struct {
	Symbol    string `gorm:"primarykey"` // token symbol
	Decimals  int
	Price     float64 // unit is AR
	ManualSet bool    // manual set
	UpdatedAt time.Time
}

type ArFee struct { // unit is winston
	ID        uint `gorm:"primarykey"`
	UpdatedAt time.Time
	Base      int64
	PerChunk  int64
}
