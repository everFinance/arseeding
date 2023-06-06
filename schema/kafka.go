package schema

import "github.com/everFinance/goar/types"

type KafkaOrderInfo struct {
	ID       uint
	ItemId   string `json:"itemId"`
	Signer   string `json:"signer"` // item signer
	SignType int    `json:"signType"`
	Size     int64  `json:"size"`
}

type KafkaBundleItem struct {
	SignatureType int         `json:"signatureType"`
	Signature     string      `json:"signature"`
	Owner         string      `json:"owner"`  //  utils.Base64Encode(pubkey)
	Target        string      `json:"target"` // optional, if exist must length 32, and is base64 str
	Anchor        string      `json:"anchor"` // optional, if exist must length 32, and is base64 str
	Tags          []types.Tag `json:"tags"`
	Id            string      `json:"id"`

	Size    int64  `json:"size"`
	Address string `json:"address"`
	Type    string `json:"type"` // data type
}

type KafkaOnChainInfo struct {
	BundleIn  string   `json:"bundleIn"`
	ItemIds   []string `json:"itemIds"`
	Id        string   `json:"id"` // blockId
	Height    int64    `json:"height"`
	Timestamp int64    `json:"timestamp"`
	Previous  string   `json:"previous"`
}
