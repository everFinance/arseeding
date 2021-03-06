package schema

type FeeConfig struct {
	SpeedTxFee     int64 `json:"speedTxFee"`
	BundleServeFee int64 `json:"bundleServeFee"`
}

type IpRateWhitelist struct {
	OriginOrIP  string // e.g "188.0.2.2"
	Available   bool   `gorm:"index:idx1"` // true means effective
	Description string
}

type ApiKey struct {
	Key         string
	Description string
}
