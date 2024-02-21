package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	arseedSchema "github.com/everFinance/arseeding/schema"
	"github.com/everFinance/arseeding/sdk/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	paySchema "github.com/everVision/everpay-kits/schema"
	paySdk "github.com/everVision/everpay-kits/sdk"
	"math/big"
	"os"
	"time"
)

type SDK struct {
	ItemSigner *goar.ItemSigner
	Cli        *ArSeedCli
	Pay        *paySdk.SDK
}

func NewSDK(arseedUrl, payUrl string, signer interface{}) (*SDK, error) {
	cli := New(arseedUrl)
	itemSigner, err := goar.NewItemSigner(signer)
	if err != nil {
		return nil, err
	}
	pay, err := paySdk.New(signer, payUrl)
	if err != nil {
		return nil, err
	}
	return &SDK{
		ItemSigner: itemSigner,
		Cli:        cli,
		Pay:        pay,
	}, nil
}

func (s *SDK) SendDataAndPay(data []byte, currency string, option *schema.OptionItem, needSequence bool, rewards []schema.Reward) (everTx *paySchema.Transaction, itemId string, err error) {
	order, err := s.SendData(data, currency, "", option, needSequence)
	if err != nil {
		return
	}
	itemId = order.ItemId
	everTx, err = s.PayOrders([]*arseedSchema.RespOrder{order}, rewards)
	return
}

func (s *SDK) SendDataStreamAndPay(data *os.File, currency string, option *schema.OptionItem, needSequence bool, rewards []schema.Reward) (everTx *paySchema.Transaction, itemId string, err error) {
	order, err := s.SendDataStream(data, currency, "", option, needSequence)
	if err != nil {
		return
	}
	itemId = order.ItemId
	everTx, err = s.PayOrders([]*arseedSchema.RespOrder{order}, rewards)
	return
}

func (s *SDK) SendData(data []byte, currency string, apikey string, option *schema.OptionItem, needSequence bool) (order *arseedSchema.RespOrder, err error) {
	bundleItem := types.BundleItem{}
	if option != nil {
		bundleItem, err = s.ItemSigner.CreateAndSignItem(data, option.Target, option.Anchor, option.Tags)
	} else {
		bundleItem, err = s.ItemSigner.CreateAndSignItem(data, "", "", nil)
	}
	if err != nil {
		return
	}
	order, err = s.Cli.SubmitItem(bundleItem.ItemBinary, currency, apikey, needSequence)
	return
}

func (s *SDK) SendDataStream(data *os.File, currency string, apikey string, option *schema.OptionItem, needSequence bool) (order *arseedSchema.RespOrder, err error) {
	bundleItem := types.BundleItem{}
	if option != nil {
		bundleItem, err = s.ItemSigner.CreateAndSignItemStream(data, option.Target, option.Anchor, option.Tags)
	} else {
		bundleItem, err = s.ItemSigner.CreateAndSignItemStream(data, "", "", nil)
	}
	if err != nil {
		return
	}
	binaryReader, err := utils.GenerateItemBinaryStream(&bundleItem)
	if err != nil {
		return
	}
	order, err = s.Cli.SubmitItemStream(binaryReader, currency, apikey, needSequence)
	return
}

func (s *SDK) BatchPayOrders(orders []*arseedSchema.RespOrder, rewards []schema.Reward) (everTxs []*paySchema.Transaction, err error) {
	if len(orders) <= 500 {
		everTx, err := s.PayOrders(orders, rewards)
		if err != nil {
			return nil, err
		}
		return []*paySchema.Transaction{everTx}, nil
	}

	// more than 500
	start := 0
	end := 500
	for {
		subOrders := orders[start:end]
		everTx, err := s.PayOrders(subOrders, rewards)
		if err != nil {
			return nil, err
		}
		everTxs = append(everTxs, everTx)

		start = end
		end += 500
		if end > len(orders) {
			end = len(orders)
		}
		if start == end {
			break
		}
	}
	return
}

func (s *SDK) PayOrders(orders []*arseedSchema.RespOrder, rewards []schema.Reward) (everTx *paySchema.Transaction, err error) {
	if len(orders) == 0 {
		return nil, errors.New("order is null")
	}
	if orders[0].Fee == "" { // arseeding NO_FEE module
		return
	}

	// orders can not more than 500
	if len(orders) > 500 {
		return nil, errors.New("please use BatchPayOrders function")
	}

	// check orders
	if len(orders) > 1 {
		bundler := orders[0].Bundler
		currency := orders[0].Currency
		for _, ord := range orders[1:] {
			if ord.Bundler != bundler || ord.Currency != currency {
				return nil, errors.New("orders bundler and currency must be equal")
			}
		}
	}
	totalFee := big.NewInt(0)
	itemIds := make([]string, 0, len(orders))
	for _, ord := range orders {
		feeInt, ok := new(big.Int).SetString(ord.Fee, 10)
		if !ok {
			return nil, errors.New("order fee incorrect")
		}
		totalFee = new(big.Int).Add(totalFee, feeInt)
		itemIds = append(itemIds, ord.ItemId)
	}

	payTxData := arseedSchema.PaymentMeta{
		AppName: "arseeding",
		Action:  "payment",
		ItemIds: itemIds,
	}
	dataJs, err := json.Marshal(&payTxData)
	if err != nil {
		return
	}
	tokenTags := s.Pay.SymbolToTagArr(orders[0].Currency)
	if len(tokenTags) == 0 {
		err = errors.New("currency not exist token")
		return
	}
	tokBals, err := s.Pay.Cli.Balances(s.Pay.AccId)
	if err != nil {
		return
	}
	tagToBal := make(map[string]*big.Int)
	for _, bal := range tokBals.Balances {
		amt, _ := new(big.Int).SetString(bal.Amount, 10)
		tagToBal[bal.Tag] = amt
	}
	useTag := ""
	for _, tag := range tokenTags {
		amt, ok := tagToBal[tag]
		if !ok {
			continue
		}
		if amt.Cmp(totalFee) >= 0 {
			useTag = tag
		}
	}
	if useTag == "" {
		err = errors.New("token balance insufficient")
		return
	}

	if len(rewards) > 0 {
		// sdk need 10% reward
		rewards = append(rewards, schema.Reward{
			Tag:       useTag,
			Recipient: "0x28840a3A565eF1fE7468d000Ff0B461058b9Ef5f",
			Amount:    new(big.Int).Div(totalFee, big.NewInt(10)).String(), // 10%
		})

		var bundleSigs paySchema.BundleWithSigs
		bundleSigs, err = s.assembleEverBundleItems(rewards)
		if err != nil {
			return
		}
		everTx, err = s.Pay.BundleWithData(useTag, orders[0].Bundler, totalFee, bundleSigs, string(dataJs))
	} else {
		everTx, err = s.Pay.Transfer(useTag, totalFee, orders[0].Bundler, string(dataJs))
	}
	return
}

func (s *SDK) PayApikey(tokenTag string, amount *big.Int) (everHash string, err error) {
	bundler, err := s.Cli.GetBundler()
	if err != nil {
		return
	}
	payTxData := struct {
		AppName string `json:"appName"`
		Action  string `json:"action"`
		Bundler string `json:"bundler"` // option
	}{
		AppName: "arseeding",
		Action:  "apikeyPayment",
		Bundler: bundler,
	}
	dataJs, err := json.Marshal(payTxData)
	if err != nil {
		return
	}

	everTx, err := s.Pay.Transfer(tokenTag, amount, bundler, string(dataJs))
	if err != nil {
		fmt.Println("2")
		return
	}
	everHash = everTx.HexHash()
	return
}

func (s *SDK) assembleEverBundleItems(rewards []schema.Reward) (paySchema.BundleWithSigs, error) {
	items := make([]paySchema.BundleItem, 0, len(rewards))
	tokens := s.Pay.GetTokens()
	for _, rd := range rewards {
		tok, ok := tokens[rd.Tag]
		if !ok {
			return paySchema.BundleWithSigs{}, errors.New("token tag not exist")
		}
		items = append(items, paySchema.BundleItem{
			Tag:     rd.Tag,
			ChainID: tok.ChainID,
			From:    s.Pay.AccId,
			To:      rd.Recipient,
			Amount:  rd.Amount,
		})
	}
	bundleWithoutSig := paySdk.GenBundle(items, time.Now().Unix()+100)
	return s.Pay.SignBundleData(bundleWithoutSig)
}
