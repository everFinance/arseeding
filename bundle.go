package arseeding

import (
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/go-everpay/account"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"math"
	"strings"
	"time"
)

func (s *Arseeding) ProcessSubmitItem(item types.BundleItem, currency string, isNoFeeMode bool, apiKey string, isSort bool, size int64) (schema.Order, error) {
	if err := utils.VerifyBundleItem(item); err != nil {
		return schema.Order{}, err
	}
	if item.DataReader != nil { // reset io stream to origin of the file
		if _, err := item.DataReader.Seek(0, 0); err != nil {
			return schema.Order{}, err
		}
	}
	// store item
	if err := s.saveItem(item); err != nil {
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
		ItemId:        item.Id,
		Signer:        accId,
		SignType:      item.SignatureType,
		Size:          size,
		ExpectedBlock: s.cache.GetInfo().Height + s.expectedRange,
		OnChainStatus: schema.WaitOnChain,
		ApiKey:        apiKey,
		Sort:          isSort,
	}
	// calc fee
	respFee, err := s.CalcItemFee(currency, order.Size)
	if err != nil {
		return schema.Order{}, err
	}
	order.Decimals = respFee.Decimals
	order.Fee = respFee.FinalFee
	order.Currency = strings.ToUpper(currency)

	if isNoFeeMode {
		order.PaymentStatus = schema.SuccPayment
	} else {
		order.PaymentExpiredTime = time.Now().Unix() + s.paymentExpiredRange
		order.PaymentStatus = schema.UnPayment
	}

	// insert to mysql
	if err = s.wdb.InsertOrder(order); err != nil {
		return schema.Order{}, err
	}
	return order, nil
}

func (s *Arseeding) CalcItemFee(currency string, itemSize int64) (*schema.RespFee, error) {
	perFee := s.GetPerFee(currency)
	if perFee == nil {
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
	arPrice, err := s.wdb.GetArPrice()
	if err != nil {
		return nil, err
	}
	tps, err := s.wdb.GetPrices()
	if err != nil {
		return nil, err
	}
	arFee := s.cache.GetFee()
	arFee.Base = arFee.Base + s.config.GetServeFee()         // add base arseeding service fee
	arFee.PerChunk = arFee.PerChunk + s.config.GetServeFee() // add base arseeding service fee
	res := make(map[string]schema.Fee)
	for _, tp := range tps {
		if tp.Price <= 0.0 {
			continue
		}

		// fee = 1e(tpDecimals) * arPrice * arBaseFee / 1e(arDeciamls) / tpPrice
		baseFee := decimal.NewFromFloat(math.Pow10(tp.Decimals)).Mul(decimal.NewFromFloat(arPrice)).Mul(decimal.NewFromInt(arFee.Base)).
			Div(decimal.NewFromFloat(math.Pow10(12))).Div(decimal.NewFromFloat(tp.Price)).Round(0)

		perChunkFee := decimal.NewFromFloat(math.Pow10(tp.Decimals)).Mul(decimal.NewFromFloat(arPrice)).Mul(decimal.NewFromInt(arFee.PerChunk)).
			Div(decimal.NewFromFloat(math.Pow10(12))).Div(decimal.NewFromFloat(tp.Price)).Round(0)

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
		// process manifest
		if s.EnableManifest && getTagValue(item.Tags, schema.ContentType) == schema.ManifestType {
			mfUrl := expectedTxSandbox(item.Id)
			if _, err = s.wdb.GetManifestId(mfUrl); err == gorm.ErrRecordNotFound {
				// insert new record
				if err = s.wdb.InsertManifest(schema.Manifest{
					ManifestUrl: mfUrl,
					ManifestId:  item.Id,
				}); err != nil {
					log.Error("s.wdb.InsertManifest(res)", "err", err, "itemId", item.Id)
					return err
				}
			}
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
	return s.store.AtomicSaveItem(item)
}

func (s *Arseeding) DelItem(itemId string) error {
	if !s.store.IsExistItemBinary(itemId) {
		return nil
	}

	return s.store.AtomicDelItem(itemId)
}
