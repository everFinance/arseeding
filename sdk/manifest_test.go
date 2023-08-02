package sdk

import (
	"github.com/everFinance/goether"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSDK_UploadFolderAndPay(t *testing.T) {
	priKey := "1d8bdd0d2f1e73dffe1111111118325b7e195669541f76559760ef615a588be3"
	eccSigner, err := goether.NewSigner(priKey)
	assert.NoError(t, err)
	seedUrl := "https://seed-dev.everpay.io"
	payUrl := "https://api.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, eccSigner)
	assert.NoError(t, err)

	rootPath := "./dist"
	orders, manifestId, everTxs, err := sdk.UploadFolderAndPay(rootPath, 20, "index.html", "usdt")
	assert.NoError(t, err)
	t.Log(len(orders))
	t.Log("manifestId:", manifestId)
	t.Log("everTx:", everTxs[0].HexHash())
}

func TestSDK_UploadFolder(t *testing.T) {
	priKey := "1d8bdd0d2f1e73dffe1111111118325b7e195669541f76559760ef615a588be3"
	eccSigner, err := goether.NewSigner(priKey)
	assert.NoError(t, err)
	seedUrl := "https://arseed-dev.web3infra.dev"
	payUrl := "https://api.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, eccSigner)
	assert.NoError(t, err)

	rootPath := "./dist"
	orders, manifestId, err := sdk.UploadFolder(rootPath, 20, "index.html", "usdt")
	assert.NoError(t, err)
	t.Log(len(orders))
	t.Log("manifestId:", manifestId)
	// pay fee
	everTxs, err := sdk.BatchPayOrders(orders)
	t.Log("everTx:", everTxs[0].HexHash())
}

func TestSDK_UploadFolderWithNoFee(t *testing.T) {
	priKey := "1d8bdd0d2f1e73dffe1111111118325b7e195669541f76559760ef615a588be3"
	eccSigner, err := goether.NewSigner(priKey)
	assert.NoError(t, err)
	seedUrl := "https://arseed.web3infra.dev"
	payUrl := "https://api.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, eccSigner)
	assert.NoError(t, err)
	apikey := "xxxxxxxxxx"

	rootPath := "./build"
	orders, manifestId, err := sdk.UploadFolderWithNoFee(rootPath, 20, "index.html", apikey)
	assert.NoError(t, err)
	t.Log(len(orders))
	t.Log("manifestId:", manifestId)
}
