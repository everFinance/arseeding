package arseeding

import (
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/everpay/account"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"
	"time"
)

func (s *Arseeding) ProcessSubmitItem(item types.BundleItem, currency string) (schema.Order, error) {
	if err := utils.VerifyBundleItem(item); err != nil {
		return schema.Order{}, err
	}

	// store item
	if err := s.saveItem(item); err != nil {
		return schema.Order{}, err
	}

	// calc fee
	size := int64(len(item.ItemBinary))
	respFee, err := s.CalcItemFee(currency, size)
	if err != nil {
		return schema.Order{}, err
	}
	signerAddr, err := utils.ItemSignerAddr(item)
	if err != nil {
		return schema.Order{}, err
	}
	_, accId, err := account.IDCheck(signerAddr)
	if err != nil {
		return schema.Order{}, err
	}
	order := schema.Order{
		ItemId:             item.Id,
		Signer:             accId,
		SignType:           item.SignatureType,
		Size:               size,
		Currency:           strings.ToUpper(currency),
		Decimals:           respFee.Decimals,
		Fee:                respFee.FinalFee,
		PaymentExpiredTime: time.Now().Unix() + s.paymentExpiredRange,
		ExpectedBlock:      s.arInfo.Height + s.expectedRange,
		PaymentStatus:      schema.UnPayment,
		PaymentId:          "",
		OnChainStatus:      schema.WaitOnChain,
	}

	// insert to mysql
	if err = s.wdb.InsertOrder(order); err != nil {
		return schema.Order{}, err
	}
	return order, nil
}

func (s *Arseeding) CalcItemFee(currency string, itemSize int64) (*schema.RespFee, error) {
	perFee, ok := s.bundlePerFeeMap[strings.ToUpper(currency)]
	if !ok {
		return nil, fmt.Errorf("not support currency: %s", currency)
	}

	count := int64(0)
	if itemSize > 0 {
		count = (itemSize-1)/types.MAX_CHUNK_SIZE + 1
	}

	chunkFees := decimal.NewFromInt(count).Mul(perFee.PerChunk)
	finalFee := perFee.Base.Add(chunkFees)

	return &schema.RespFee{
		Currency: perFee.Currency,
		Decimals: perFee.Decimals,
		FinalFee: finalFee.String(),
	}, nil
}

func (s *Arseeding) GetBundlePerFees() (map[string]schema.Fee, error) {
	tps, err := s.wdb.GetPrices()
	if err != nil {
		return nil, err
	}
	arFee := s.cache.GetFee()
	res := make(map[string]schema.Fee)
	for _, tp := range tps {
		if tp.Price <= 0.0 {
			continue
		}
		price := decimal.NewFromBigInt(utils.ARToWinston(big.NewFloat(tp.Price)), 0)
		baseFee := decimal.NewFromInt(arFee.Base).Div(price)
		perChunkFee := decimal.NewFromInt(arFee.PerChunk).Div(price)
		res[strings.ToUpper(tp.Symbol)] = schema.Fee{
			Currency: tp.Symbol,
			Decimals: tp.Decimals,
			Base:     baseFee,
			PerChunk: perChunkFee,
		}
	}
	return res, nil
}

func (s *Arseeding) ParseAndSaveBundleItems(arId string, data []byte) error {
	if s.store.ExistArIdToItemIds(arId) {
		return nil
	}

	bundle, err := utils.DecodeBundle(data)
	if err != nil {
		return err
	}
	itemIds := make([]string, 0, len(bundle.Items))
	// save items
	for _, item := range bundle.Items {
		if err = s.saveItem(item); err != nil {
			log.Error("s.saveItem(item)", "err", err, "arId", arId)
			return err
		}
		itemIds = append(itemIds, item.Id)
	}

	// save arId to itemIds
	return s.store.SaveArIdToItemIds(arId, itemIds)
}

func (s *Arseeding) saveItem(item types.BundleItem) error {
	if s.store.IsExistItemBinary(item.Id) {
		return nil
	}

	boltTx, err := s.store.BoltDb.Begin(true)
	if err != nil {
		log.Error("s.store.BoltDb.Begin(true)", "err", err)
		return err
	}

	if err = s.store.SaveItemBinary(item.Id, item.ItemBinary, boltTx); err != nil {
		boltTx.Rollback()
		log.Error("saveItemBinary failed", "err", err, "itemId", item.Id)
		return err
	}

	if err = s.store.SaveItemMeta(item, boltTx); err != nil {
		boltTx.Rollback()
		return err
	}
	// commit
	if err = boltTx.Commit(); err != nil {
		boltTx.Rollback()
		return err
	}
	return nil
}
