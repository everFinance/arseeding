package sdk

import (
	"github.com/everFinance/arseeding/sdk/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goether"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSDK_SendDataAndPay_EccSigner(t *testing.T) {
	priKey := "9d8bdd0d2f1e73dffe1111111118325b7e195669541f76559760ef615a588be0"
	eccSigner, err := goether.NewSigner(priKey)
	if err != nil {
		panic(err)
	}
	payUrl := "https://api-dev.everpay.io"
	seedUrl := "https://seed-dev.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, eccSigner)
	if err != nil {
		panic(err)
	}
	data := []byte("some data")
	tags := []types.Tag{
		{"Content-Type", "text"},
	}
	tx, itemId, err := sdk.SendDataAndPay(data, "usdt", &schema.OptionItem{Tags: tags}) // your account must have enough balance in everpay
	assert.NoError(t, err)
	t.Log("itemId:", itemId)
	t.Log(tx.HexHash())
}

func TestSDK_SendDataAndPay_RsaSigner(t *testing.T) {
	payUrl := "https://api-dev.everpay.io"
	seedUrl := "https://seed-dev.everpay.io"

	rsaSigner, err := goar.NewSignerFromPath("./rsakey.json")
	if err != nil {
		panic(err)
	}
	sdk, err := NewSDK(seedUrl, payUrl, rsaSigner)
	if err != nil {
		panic(err)
	}
	data := []byte("some data")
	tags := []types.Tag{
		{"Content-Type", "text"},
	}
	tx, itemId, err := sdk.SendDataAndPay(data, "usdt", &schema.OptionItem{Tags: tags}) // your account must have enough balance in everpay
	assert.NoError(t, err)
	t.Log("itemId:", itemId)
	t.Log(tx.HexHash())
}

func TestArSeedCli_SubmitNativeData(t *testing.T) {
	apiKey := "aabbccddeee"
	data := []byte("aaabbbcc")
	cli := New("http://127.0.0.1:8080")

	res, err := cli.SubmitNativeData(apiKey, data, "image/jpeg", map[string]string{
		"key1": "arseeding test",
		"key2": "sandy test bundle native data",
	})
	assert.NoError(t, err)
	t.Log(res)

}
