package bundle_item

import (
	"github.com/everFinance/arseeding/sdk"
	"github.com/everFinance/goar"
	"github.com/everFinance/goether"
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
	url := "https://seed-dev.everpay.io" // your arseeding service address
	arseedSdk := sdk.New(url)

	// create bundle item with goar
	data := []byte("your data") // your data,maybe read from files
	item, err := itemSigner.CreateAndSignItem(data, "", "", nil)
	if err != nil {
		panic(err)
	}

	// send bundle item to arseeding with arseeding sdk
	res, err := arseedSdk.SubmitItem(item.ItemBinary, "USDC") // use "USDC" token payment fee
	if err != nil {
		t.Log(err)
	}
	t.Log(res)
}
