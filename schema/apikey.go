package schema

import (
	"gorm.io/datatypes"
	"time"
)

type AutoApiKey struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	ApiKey       string `gorm:"index:apikey01,unique"`
	EncryptedKey string

	Address      string `gorm:"index:apikey02,unique"`
	PubKey       string
	TokenBalance datatypes.JSONMap // key: symbol,val: balance
}
