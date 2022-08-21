package sdk

import (
	"github.com/everFinance/goether"
	"testing"
)

func TestSDK_UploadFolder(t *testing.T) {
	priKey := "1d8bdd0d2f1e73dffe1111111118325b7e195669541f76559760ef615a588be3"
	eccSigner, err := goether.NewSigner(priKey)
	if err != nil {
		panic(err)
	}
	seedUrl := "http://127.0.0.1:8080"
	payUrl := "https://api-dev.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, eccSigner)
	if err != nil {
		panic(err)
	}

	rootPath := "./dist"
	orders, err := sdk.UploadFolder(rootPath, 20, "", "usdt")
	if err != nil {
		panic(err)
	}
	t.Log(orders[len(orders)-1].ItemId)
	t.Log(len(orders))

	// pay orders
}
