package schema

import "gorm.io/gorm"

type ApiKey struct {
	gorm.Model
	Key          string `json:"apiKey"`
	PubKey       string `json:"pubKey"`
	Address      string `json:"address"`
	EncryptedKey string `json:"encryptedApikey"`
	EverHash     string `json:"everHash"`
	Cap          int64  `json:"cap"`
}

type RegisterResp struct {
	Key string `json:"apikey"`
	Cap int64  `json:"cap"`
}

type ExpandResp struct {
	Cap int64 `json:"cap"`
}

type HeldApiKeys struct {
	EncryptedKey string `json:"encryptedKey"`
	Cap          int64  `json:"cap"`
}

type ExpandRecord struct {
	gorm.Model
	ParentHash string
	ChildHash  string
}
