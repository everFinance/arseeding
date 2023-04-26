package sdk

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/everFinance/arseeding/sdk/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goether"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
)

func TestNewSDK(t *testing.T) {
	k, _ := crypto.GenerateKey()
	t.Log(k.Public())
	t.Log(crypto.PubkeyToAddress(k.PublicKey).String())
	t.Log(hex.EncodeToString(crypto.FromECDSAPub(&k.PublicKey)))
	t.Log(hex.EncodeToString(crypto.FromECDSA(k)))
	pubkey := hex.EncodeToString(crypto.FromECDSAPub(&k.PublicKey))
	pub, err := hex.DecodeString(pubkey)
	assert.NoError(t, err)
	p, err := crypto.UnmarshalPubkey(pub)
	assert.NoError(t, err)
	t.Log(crypto.PubkeyToAddress(*p).String())
}

func TestSDK_SendDataAndPay_RsaSigner(t *testing.T) {
	payUrl := "https://api.everpay.io"
	seedUrl := "https://seed-dev.everpay.io"

	rsaSigner, err := goar.NewSignerFromPath("./rsakey.json")
	if err != nil {
		panic(err)
	}
	sdk, err := NewSDK(seedUrl, payUrl, rsaSigner)
	if err != nil {
		panic(err)
	}
	data := []byte("aabbcc")
	tags := []types.Tag{
		{"Content-Type", "text"},
	}
	tx, itemId, err := sdk.SendDataAndPay(data, "usdt", &schema.OptionItem{Tags: tags}, false) // your account must have enough balance in everpay
	assert.NoError(t, err)
	t.Log("itemId:", itemId)
	t.Log(tx.HexHash())
}

func TestSDK_SendData_EccSigner(t *testing.T) {
	priKey := "2b8258cde747e3820e56a40aec5cd473150c6078819b45afe61baaf1fa1c75e6" // for test
	eccSigner, err := goether.NewSigner(priKey)
	if err != nil {
		panic(err)
	}

	payUrl := "https://api.everpay.io"
	seedUrl := "https://seed-dev.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, eccSigner)
	if err != nil {
		panic(err)
	}
	tags := []types.Tag{
		{"Content-Type", "video/mpeg4"},
	}
	data, err := ioutil.ReadFile("")
	// data,err := ioutil.ReadFile("/Users/sandyzhou/Downloads/在北海道的大雪里露营.mp4")
	assert.NoError(t, err)
	apikey := "7268f534-c866-11ed-b467-cedcf45dbf05" // for test
	t.Log("aaaa")
	ord, err := sdk.SendData(data, "usdc", apikey, &schema.OptionItem{Tags: tags}, false) // your account must have enough balance in everpay
	assert.NoError(t, err)
	t.Log("itemId:", ord.ItemId)
}

func TestArSeedCli_SubmitNativeData(t *testing.T) {
	apiKey := "8cedb476-c7c9-11ed-a52b-22b1cc528926"
	data := []byte("bbbbbbbbaaadf")
	cli := New("https://seed-dev.everpay.io")
	currency := "DODO"
	res, err := cli.SubmitNativeData(apiKey, currency, data, "image/jpeg", map[string]string{
		"key1": "arseeding test",
		"key2": "sandy test bundle native data",
	})
	assert.NoError(t, err)
	t.Log(res)
}

func TestSDK_PayApikey(t *testing.T) {
	priKey := ""
	eccSigner, err := goether.NewSigner(priKey)
	if err != nil {
		panic(err)
	}
	payUrl := "https://api.everpay.io"
	seedUrl := "https://seed-dev.everpay.io"
	sdk, err := NewSDK(seedUrl, payUrl, eccSigner)
	if err != nil {
		panic(err)
	}
	tokenSymbol := "GLMR"
	amount := big.NewInt(500000000000000000)
	tokenTags := sdk.Pay.SymbolToTagArr(tokenSymbol)
	if len(tokenTags) == 0 {
		panic("symbol not exist")
	}
	everHash, err := sdk.PayApikey(tokenTags[0], amount)
	assert.NoError(t, err)
	t.Log(everHash)
}

func TestDecryptoApikey(t *testing.T) {
	priKey := ""
	eccSigner, err := goether.NewSigner(priKey)
	if err != nil {
		panic(err)
	}
	encKey := "041fb60718c9e3d0b4be5cd746945efdac698d41c1b0cec4fe54e4cdfaeb4c8576f447c69f9e6f16b4e5a706cb176c61934e173df4c951823ec9eab97d355f91ced5f0cbd9a4a086fdb993f69520f01b06adcf505ea16b18a64be4940d519a6cfb89afe0ae709ecae1389043acfa0fa38ac763e3ae3fe354347cddba6e5eecbdffec4d106f85cfcd76fa7cc7d7f61ddd7594967f38"
	apikey, err := eccSigner.Decrypt(common.Hex2Bytes(encKey))
	assert.NoError(t, err)
	assert.Equal(t, "8cedb476-c7c9-11ed-a52b-22b1cc528926", string(apikey))
}

func TestArSeedCli_SubmitNativeDataStream(t *testing.T) {
	apiKey := "aabbccddeee"
	data, err := os.Open("fileName")
	assert.NoError(t, err)
	cli := New("http://127.0.0.1:8080")

	res, err := cli.SubmitNativeDataStream(apiKey, "usdc", data, "image/jpeg", map[string]string{
		"key1": "arseeding test",
		"key2": "sandy test bundle native data",
	})
	assert.NoError(t, err)
	t.Log(res)
}

func TestSDK_SendDataStreamAndPay(t *testing.T) {
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
	data, err := os.Open("fileName")
	assert.NoError(t, err)
	tags := []types.Tag{
		{"Content-Type", "text"},
	}
	tx, itemId, err := sdk.SendDataStreamAndPay(data, "usdt", &schema.OptionItem{Tags: tags}, false) // your account must have enough balance in everpay
	assert.NoError(t, err)
	t.Log("itemId:", itemId)
	t.Log(tx.HexHash())
}
