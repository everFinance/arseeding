package schema

type FeeConfig struct {
	SpeedTxFee        int64  `json:"speedTxFee"`
	BundleServeFee    int64  `json:"bundleServeFee"`
	FeeCollectAddress string `json:"feeCollectAddress"` // fee collection address
}

type IpRateWhitelist struct {
	OriginOrIP  string // e.g "188.0.2.2"
	Available   bool   `gorm:"index:idx3"` // true means effective
	Description string
}

type Param struct {
	ChunkConcurrentNum int
}
