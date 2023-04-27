package bundle_item

import (
	"encoding/json"
	"github.com/everFinance/arseeding/sdk"
	paySdk "github.com/everFinance/go-everpay/sdk"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goether"
	"io/ioutil"
	"math/big"
	"sync"
	"testing"
)

func TestSendItem(t *testing.T) {
	priv := "1f534ac18009182c07d266fe4a7903c0bcc8a66190f0967b719b2b3974a69c2f"
	eccSigner, err := goether.NewSigner(priv)
	if err != nil {
		panic(err)
	}
	t.Log(eccSigner.Address)

	itemSigner, err := goar.NewItemSigner(eccSigner)
	if err != nil {
		panic(err)
	}

	arseedSdk := sdk.New("https://seed-dev.everpay.io")
	payCli, err := paySdk.New(eccSigner, "https://api-dev.everpay.io")
	if err != nil {
		panic(err)
	}

	// load need upload file path array
	filePath, err := ioutil.ReadFile("./filepath.json")
	if err != nil {
		panic(err)
	}
	pathArr := make([]string, 0, 10000)
	err = json.Unmarshal(filePath, &pathArr)
	if err != nil {
		panic(err)
	}
	t.Log(len(pathArr))

	// upload bundle item to arseed serve
	var wg sync.WaitGroup
	for idx, pt := range pathArr[2606:2607] {
		data, err := ioutil.ReadFile(pt)
		if err != nil {
			t.Log("ReadFile failed", "idx", idx, "err", err)
			continue
		}
		wg.Add(1)
		go func(data []byte, idx int) {
			defer wg.Done()
			// assemble bundle item
			item, err := itemSigner.CreateAndSignItem(data, "", "", []types.Tag{
				{Name: "Content-Type", Value: "text/html"},
				{Name: "Contributor", Value: "chainnews-archive.org"},
				{Name: "App-Name", Value: "everFinance-arseeding"},
			})
			if err != nil {
				t.Log("send failed", "idx", idx, "err", err)
				return
			}
			// submit to arseed
			order, err := arseedSdk.SubmitItem(item.ItemBinary, "usdt", "", false)
			if err != nil {
				t.Log("send failed", "idx", idx, "err", err)
				return
			}

			// use everpay payment fee
			amount, _ := new(big.Int).SetString(order.Fee, 10)
			dataJs, err := json.Marshal(&order)
			if err != nil {
				panic(err)
			}
			tokenTags := payCli.SymbolToTagArr(order.Currency)
			_, err = payCli.Transfer(tokenTags[0], amount, order.Bundler, string(dataJs))
			if err != nil {
				t.Log("send failed", "idx", idx, "err", err)
				return
			}
		}(data, idx)
	}
	wg.Wait()
}
