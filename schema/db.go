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

	PendingPayment = "pending"
	SuccPayment    = "success"
	ExpiredPayment = "expired"
	FailedPayment  = "failed"
)

type Order struct {
	gorm.Model
	ItemId   string // bundleItem id
	Signer   string // item signer
	SignType int

	Currency           string // payment token symbol
	Decimals           int
	Fee                string
	PaymentExpiredTime int64
	ExpectedBlock      int64

	PaymentStatus string // "pending", "success", "expired", "failed"
	PaymentId     string // everHash

	OnChainStatus string // "waiting","pending","success","failed"
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
