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
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	ItemId   string `json:"itemId"`                   // bundleItem id
	Signer   string `gorm:"index:idx1" json:"signer"` // item signer
	SignType int    `json:"signType"`

	Size               int64  `json:"size"`
	Currency           string `json:"currency"` // payment token symbol
	Decimals           int    `json:"decimals"`
	Fee                string `json:"fee"`
	PaymentExpiredTime int64  `json:"paymentExpiredTime"` // uint s
	ExpectedBlock      int64  `json:"expectedBlock"`

	PaymentStatus string `json:"paymentStatus"` // "unpaid", "paid", "expired"
	PaymentId     string `json:"paymentId"`     // everHash

	OnChainStatus string `json:"onChainStatus"` // "waiting","pending","success","failed"
	ApiKey        string `gorm:"index:idx2" json:"-"`
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
	DataSize  string
	Reward    string         // onchain arTx reward
	Status    string         // "pending","success"
	ItemIds   datatypes.JSON // json.marshal(itemIds)
	ItemNum   int
}
