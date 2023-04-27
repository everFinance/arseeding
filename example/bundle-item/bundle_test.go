package bundle_item

import (
	"encoding/json"
	"github.com/everFinance/arseeding/sdk"
	paySdk "github.com/everFinance/go-everpay/sdk"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goether"
	"math/big"
	"testing"
)

func TestItemUseCase(t *testing.T) {
	// create a eth signer for sign bundle item
	priv := "88834ac18009182c07d116fe4a7903c0bcc8a66190f0967b719b2b3974a69c2f" // your eth private key
	eccSigner, err := goether.NewSigner(priv)
	if err != nil {
		panic(err)
	}
	t.Log(eccSigner.Address)
	itemSigner, err := goar.NewItemSigner(eccSigner)
	if err != nil {
		panic(err)
	}

	// create Arseeding SDK
	url := "https://arseed.web3infura.io" // your arseeding service address
	arseedSdk := sdk.New(url)

	// create bundle item with goar
	data := []byte("your data") // your data,maybe read from files
	item, err := itemSigner.CreateAndSignItem(data, "", "", []types.Tag{})
	if err != nil {
		panic(err)
	}

	// send bundle item to arseeding with arseeding sdk
	order, err := arseedSdk.SubmitItem(item.ItemBinary, "USDC", "", false) // use "USDC" token payment fee
	if err != nil {
		t.Log(err)
	}
	t.Log(order)

	// use everpay payment fee
	amount, _ := new(big.Int).SetString(order.Fee, 10)
	dataJs, err := json.Marshal(&order)
	if err != nil {
		panic(err)
	}
	payCli, err := paySdk.New(eccSigner, "https://api.everpay.io")
	if err != nil {
		panic(err)
	}
	tokenTags := payCli.SymbolToTagArr(order.Currency)
	_, err = payCli.Transfer(tokenTags[0], amount, order.Bundler, string(dataJs))
	if err != nil {
		t.Log("send failed", "err", err)
		return
	}
}
