package schema

const (
	DefaultPaymentExpiredRange = int64(2592000) // 30 days
	DefaultExpectedRange       = 50             // block height range
)

type PaymentMeta struct {
	AppName string   `json:"appName"`
	Action  string   `json:"action"`
	ItemIds []string `json:"itemIds"`
}
