package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestBundleCalcItemFee(t *testing.T) {
	size0 := int64(types.MAX_CHUNK_SIZE - 1)
	size1 := int64(types.MAX_CHUNK_SIZE + 1)
	size2 := int64(types.MAX_CHUNK_SIZE)
	size3 := int64(0)
	perFeeMap := map[string]schema.Fee{
		"AR": {
			Currency: "AR",
			Decimals: 0,
			Base:     decimal.New(5, 0),
			PerChunk: decimal.New(1, 0),
		},
	}
	aa := &Arseeding{
		bundlePerFeeMap: perFeeMap,
	}
	res, err := aa.CalcItemFee("AR", size0)
	assert.NoError(t, err)
	assert.Equal(t, "6", res.FinalFee)

	res, err = aa.CalcItemFee("AR", size1)
	assert.NoError(t, err)
	assert.Equal(t, "7", res.FinalFee)

	res, err = aa.CalcItemFee("AR", size2)
	assert.NoError(t, err)
	assert.Equal(t, "6", res.FinalFee)

	res, err = aa.CalcItemFee("AR", size3)
	assert.NoError(t, err)
	assert.Equal(t, "5", res.FinalFee)
}

func TestSaveDelItem(t *testing.T) {
	dbPath := "./data/tmp.db"
	signer, err := goar.NewSignerFromPath("./test-keyfile.json") // your key file path
	assert.NoError(t, err)
	bundlSigner, err := goar.NewItemSigner(signer)
	assert.NoError(t, err)
	item, err := bundlSigner.CreateAndSignItem([]byte("data 01"), "", "", nil)
	assert.NoError(t, err)
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	aa := &Arseeding{
		store: s,
	}
	err = aa.saveItem(item)
	assert.NoError(t, err)
	err = aa.DelItem(item.Id)
	assert.NoError(t, err)
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestParseAndSaveBundleItems(t *testing.T) {

	arId := "p5lopWlVbGvBiPy5kevaEWtwHHpl0coUeUv9qNQh6NA" // this arTx use bundle type
	dbPath := "./data/tmp.db"
	arNode := "https://arweave.net"
	cli := goar.NewClient(arNode)
	data, err := cli.GetTransactionData(arId)
	assert.NoError(t, err)
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	aa := &Arseeding{
		store: s,
	}
	err = aa.ParseAndSaveBundleItems(arId, data)
	assert.NoError(t, err)
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}
