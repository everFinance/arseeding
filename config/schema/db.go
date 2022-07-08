package schema

type FeeConfig struct {
	SpeedTxFee     int64 `json:"speedTxFee"`
	BundleServeFee int64 `json:"bundleServeFee"`
}
