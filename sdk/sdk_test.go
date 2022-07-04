package sdk

import (
	"github.com/everFinance/arseeding/sdk/schema"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goether"
	"testing"
)

func TestSendData(t *testing.T) {
	priKey := "9d8bdd0d2f1e73dffe1111111118325b7e195669541f76559760ef615a588be0"
	signer, err := goether.NewSigner(priKey)
	if err != nil {
		panic(err)
	}
	payUrl := "https://api-dev.everpay.io"
	seedUrl := "https://seed-dev.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, signer)
	if err != nil {
		panic(err)
	}
	data := []byte("some data")
	if err != nil {
		panic(err)
	}
	tags := []types.Tag{
		{"Content-Type", "text"},
	}
	tx, itemId, err := sdk.SendDataAndPay(data, "usdt", &schema.OptionItem{Tags: tags}) // your account must have enough balance in everpay
	t.Log(itemId)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(tx)
}
