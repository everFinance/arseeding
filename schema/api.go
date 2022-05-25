package schema

const (
	AllowMaxItemSize = 100 * 1024 * 1024 // 100 MB
)

type RespOrder struct {
	ItemId             string // bundleItem id
	Bundler            string // fee receiver address
	Currency           string // payment token symbol
	Fee                string
	PaymentExpiredTime int64
	ExpectedBlock      int64
}
