package bundle_item

import (
	"github.com/everFinance/arseeding/sdk"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goether"
	"io/ioutil"
	"testing"
)

func TestPostItemStream(t *testing.T) {
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
	url := "http://127.0.0.1:8080" // your arseeding service address
	arseedSdk := sdk.New(url)
	// data size must > maxAllowByte
	data, err := ioutil.ReadFile("img.jpeg") // your data,maybe read from files
	if err != nil {
		panic(err)
	}
	item, err := itemSigner.CreateAndSignItem(data, "", "", []types.Tag{{"Content-Type", "jpeg"}})
	if err != nil {
		panic(err)
	}

	// send bundle item to arseeding with arseeding sdk
	order, err := arseedSdk.SubmitItem(item.ItemBinary, "USDC", "", false) // use "USDC" token payment fee
	if err != nil {
		t.Log(err)
	}
	t.Log(order)
}

func TestPostNativeDataStream(t *testing.T) {
	url := "http://127.0.0.1:8080" // your arseeding service address
	arseedSdk := sdk.New(url)
	data, err := ioutil.ReadFile("img.jpeg") // your data,maybe read from files
	if err != nil {
		panic(err)
	}
	res, err := arseedSdk.SubmitNativeData("123", data, "jpeg", map[string]string{"Content-Type": "jpeg", "Name": "test"})
	if err != nil {
		panic(err)
	}
	t.Log(res.ItemId)
}
