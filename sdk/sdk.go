package sdk

import (
	"encoding/json"
	"errors"
	arseedSchema "github.com/everFinance/arseeding/schema"
	"github.com/everFinance/arseeding/sdk/schema"
	paySchema "github.com/everFinance/everpay-go/pay/schema"
	paySdk "github.com/everFinance/everpay-go/sdk"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"math/big"
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

func (s *SDK) SendDataAndPay(data []byte, currency string, option *schema.OptionItem, needSequence bool) (everTx *paySchema.Transaction, itemId string, err error) {
	order, err := s.SendData(data, currency, "", option, needSequence)
	if err != nil {
		return
	}
	itemId = order.ItemId
	everTx, err = s.PayOrders([]*arseedSchema.RespOrder{order})
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

func (s *SDK) BatchPayOrders(orders []*arseedSchema.RespOrder) (everTxs []*paySchema.Transaction, err error) {
	if len(orders) <= 500 {
		everTx, err := s.PayOrders(orders)
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
		everTx, err := s.PayOrders(subOrders)
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

func (s *SDK) PayOrders(orders []*arseedSchema.RespOrder) (everTx *paySchema.Transaction, err error) {
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

	payTxData := struct {
		AppName string   `json:"appName"`
		Action  string   `json:"action"`
		ItemIds []string `json:"itemIds"`
	}{
		AppName: "arseeding",
		Action:  "payment",
		ItemIds: itemIds,
	}
	dataJs, err := json.Marshal(&payTxData)
	if err != nil {
		return
	}
	everTx, err = s.Pay.Transfer(orders[0].Currency, totalFee, orders[0].Bundler, string(dataJs))
	return
}

func (s *SDK) PayApikey(tokenSymbol string, amount *big.Int) (everHash string, err error) {
	payTxData := struct {
		AppName string `json:"appName"`
		Action  string `json:"action"`
	}{
		AppName: "arseeding",
		Action:  "apikeyPayment",
	}
	dataJs, err := json.Marshal(&payTxData)
	if err != nil {
		return
	}
	bundler, err := s.Cli.GetBundler()
	if err != nil {
		return
	}
	everTx, err := s.Pay.Transfer(tokenSymbol, amount, bundler, string(dataJs))
	if err != nil {
		return
	}
	everHash = everTx.HexHash()
	return
}
