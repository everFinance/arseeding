package schema

import "gorm.io/gorm"

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
	ItemId string // bundleItem id
	Signer string // item signer
	SignType int

	Currency           string // payment token symbol
	Fee                string
	PaymentExpiredTime int64
	ExpectedBlock      int64

	PaymentStatus string // "pending", "success", "expired", "failed"
	PaymentId     string // everHash

	OnChainStatus string // "waiting","pending","success","failed"
}
