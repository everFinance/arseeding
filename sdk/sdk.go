package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
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

func (s *SDK) SendDataAndPay(data []byte, currency string, option *schema.OptionItem) (everTx *paySchema.Transaction, itemId string, err error) {
	order, err := s.SendData(data, currency, option)
	if err != nil {
		return
	}
	everTx, err = s.PayOrder(order)
	return
}

func (s *SDK) SendData(data []byte, currency string, option *schema.OptionItem) (order *arseedSchema.RespOrder, err error) {
	bundleItem := types.BundleItem{}
	if option != nil {
		bundleItem, err = s.ItemSigner.CreateAndSignItem(data, option.Target, option.Anchor, option.Tags)
	} else {
		bundleItem, err = s.ItemSigner.CreateAndSignItem(data, "", "", nil)
	}
	if err != nil {
		return
	}
	order, err = s.Cli.SubmitItem(bundleItem.ItemBinary, currency)
	return
}

func (s *SDK) PayOrder(order *arseedSchema.RespOrder) (everTx *paySchema.Transaction, err error) {
	if order == nil {
		return nil, errors.New("order is null")
	}
	if order.Fee == "" { // arseeding NO_FEE module
		return
	}
	amount, ok := new(big.Int).SetString(order.Fee, 10)
	if !ok {
		err = errors.New(fmt.Sprintf("new(big.Int).SetString(order.Fee, 10); fee=%s", order.Fee))
		return
	}
	dataJs, err := json.Marshal(&order)
	if err != nil {
		return
	}
	everTx, err = s.Pay.Transfer(order.Currency, amount, order.Bundler, string(dataJs))
	return
}
