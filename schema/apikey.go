package schema

import "gorm.io/gorm"

type ApiKey struct {
	gorm.Model
	Key          string `gorm:"index" json:"apiKey"`
	PubKey       string `json:"pubKey"`
	Address      string `gorm:"index" json:"address"`
	EncryptedKey string `json:"encryptedApikey"`
	EverHash     string `json:"everHash"`
	Cap          string `json:"cap"`
}

type RegisterResp struct {
	Key string `json:"apikey"`
	Cap string `json:"cap"`
}

type ExpandResp struct {
	Cap string `json:"cap"`
}

type HeldApiKeys struct {
	EncryptedKey string `json:"encryptedKey"`
	Cap          string `json:"cap"`
}

type ExpandRecord struct {
	gorm.Model
	ParentHash string `gorm:"index"`
	ChildHash  string `gorm:"index"`
}
