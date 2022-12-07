package arseeding

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/everpay-go/account"
	"github.com/everFinance/everpay-go/config"
	sdkSchema "github.com/everFinance/everpay-go/sdk/schema"
	paySchema "github.com/everFinance/everpay-go/token/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/panjf2000/ants/v2"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
	"math/big"
	"strings"
	"sync"
	"time"
)

func (s *Arseeding) runJobs(bundleInterval int) {
	// update cache
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.updateAnchor)
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.updateArFee)
	s.scheduler.Every(30).Seconds().SingletonMode().Do(s.updateInfo)
	s.scheduler.Every(5).Minute().SingletonMode().Do(s.updatePeerMap)

	// about bundle
	if !s.NoFee {
		s.scheduler.Every(5).Minute().SingletonMode().Do(s.updateTokenPrice)
		s.scheduler.Every(1).Minute().SingletonMode().Do(s.updateBundlePerFee)
		go s.watchEverReceiptTxs()
		s.scheduler.Every(5).Seconds().SingletonMode().Do(s.mergeReceiptAndOrder)
		s.scheduler.Every(2).Minute().SingletonMode().Do(s.refundReceipt)
		s.scheduler.Every(1).Minute().SingletonMode().Do(s.processExpiredOrd)
	}

	s.scheduler.Every(bundleInterval).Seconds().SingletonMode().Do(s.onChainBundleItems) // can set a longer time, if the items are less. such as 2m
	// onChainBundleItems by upload order
	s.scheduler.Every(bundleInterval).Seconds().SingletonMode().Do(s.onChainItemsBySeq)
	s.scheduler.Every(3).Minute().SingletonMode().Do(s.watchArTx)
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.retryOnChainArTx)

	s.scheduler.Every(10).Seconds().SingletonMode().Do(s.parseAndSaveBundleTx)

	// manager taskStatus
	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.watcherAndCloseTasks)

	s.scheduler.Every(1).Minute().SingletonMode().Do(s.updateBundler)

	s.scheduler.StartAsync()
}

func (s *Arseeding) updateAnchor() {
	anchor, err := fetchAnchor(s.arCli, s.cache.GetPeers())
	if err == nil {
		s.cache.UpdateAnchor(anchor)
	}
}

func (s *Arseeding) updateInfo() {
	info, err := fetchArInfo(s.arCli, s.cache.GetPeers())
	if err == nil && info != nil {
		s.cache.UpdateInfo(*info)
	}
}

func (s *Arseeding) updateArFee() {
	txPrice, err := fetchArFee(s.arCli, s.cache.GetPeers())
	if err == nil {
		s.cache.UpdateFee(txPrice)
	}
}

// update peer list concurrency, check peer available, save in db
func (s *Arseeding) updatePeerMap() {
	peers, err := s.arCli.GetPeers()
	if err != nil {
		return
	}

	availablePeers := filterPeers(peers, s.cache.GetConstTx())
	if len(availablePeers) == 0 {
		return
	}

	peerMap := updatePeerMap(s.cache.GetPeerMap(), availablePeers)

	s.cache.UpdatePeers(peerMap)
	if err = s.store.SavePeers(peerMap); err != nil {
		log.Warn("save new peer list fail")
	}
}

// bundle

func (s *Arseeding) updateTokenPrice() {
	// update symbol
	tps := make([]schema.TokenPrice, 0)
	for _, tok := range s.everpaySdk.GetTokens() {
		tps = append(tps, schema.TokenPrice{
			Symbol:    strings.ToUpper(tok.Symbol),
			Decimals:  tok.Decimals,
			ManualSet: false,
			UpdatedAt: time.Time{},
		})
	}
	if err := s.wdb.InsertPrices(tps); err != nil {
		log.Error("s.wdb.InsertPrices(tps)", "err", err)
		return
	}

	// update fee
	tps, err := s.wdb.GetPrices()
	if err != nil {
		log.Error("s.wdb.GetPrices()", "err", err)
		return
	}
	for _, tp := range tps {
		if tp.ManualSet {
			continue
		}
		price, err := config.GetTokenPriceByRedstone(tp.Symbol, "USDC")
		if err != nil {
			log.Error("config.GetTokenPriceByRedstone(tp.Symbol,\"USDC\")", "err", err, "symbol", tp.Symbol)
			continue
		}
		// update tokenPrice
		if err := s.wdb.UpdatePrice(tp.Symbol, price); err != nil {
			log.Error("s.wdb.UpdateFee(tp.Symbol,fee)", "err", err, "symbol", tp.Symbol, "fee", price)
		}
	}
}

func (s *Arseeding) updateBundlePerFee() {
	feeMap, err := s.GetBundlePerFees()
	if err != nil {
		log.Error("s.GetBundlePerFees()", "err", err)
		return
	}
	s.bundlePerFeeMap = feeMap
}

func (s *Arseeding) watcherAndCloseTasks() {
	tasks := s.taskMg.GetTasks()
	now := time.Now().Unix()
	for _, tk := range tasks {
		if tk.Close || tk.Timestamp == 0 { // timestamp == 0  means do not start
			continue
		}
		// spend time not more than 30 minutes
		if now-tk.Timestamp > 30*60 {
			if err := s.taskMg.CloseTask(tk.ArId, tk.TaskType); err != nil {
				log.Error("watcherAndCloseTasks closeJob", "err", err, "jobId", assembleTaskId(tk.ArId, tk.TaskType))
				continue
			}
		}
	}
}

func (s *Arseeding) watchEverReceiptTxs() {
	startCursor, err := s.wdb.GetLastEverRawId()
	if err != nil {
		panic(err)
	}
	subTx := s.everpaySdk.Cli.SubscribeTxs(sdkSchema.FilterQuery{
		StartCursor: startCursor,
		Address:     s.bundler.Signer.Address,
		Action:      paySchema.TxActionTransfer,
	})
	defer subTx.Unsubscribe()

	for {
		select {
		case tt := <-subTx.Subscribe():
			if tt.To != s.bundler.Signer.Address {
				continue
			}
			_, from, err := account.IDCheck(tt.From)
			if err != nil {
				log.Error("account.IDCheck(tt.From)", "err", err, "from", tt.From)
				continue
			}
			res := schema.ReceiptEverTx{
				RawId:    tt.RawId,
				EverHash: tt.EverHash,
				Nonce:    tt.Nonce,
				Symbol:   tt.TokenSymbol,
				From:     from,
				Amount:   tt.Amount,
				Data:     tt.Data,
				Status:   schema.UnSpent,
			}

			if err = s.wdb.InsertReceiptTx(res); err != nil {
				log.Error("s.wdb.InsertReceiptTx(res)", "err", err)
			}
		}
	}
}

func (s *Arseeding) mergeReceiptAndOrder() {
	unspentRpts, err := s.wdb.GetReceiptsByStatus(schema.UnSpent)
	if err != nil {
		log.Error("s.wdb.GetUnSpentReceipts()", "err", err)
		return
	}

	for _, urtx := range unspentRpts {
		itemIds, err := parseItemIds(urtx.Data)
		if err != nil {
			log.Error("parseItemIds(urtx.Data)", "err", err, "urtx", urtx.EverHash)
			if err = s.wdb.UpdateReceiptStatus(urtx.RawId, schema.UnRefund, nil); err != nil {
				log.Error("s.wdb.UpdateReceiptStatus1", "err", err, "id", urtx.RawId)
			}
			continue
		}
		// get orders by itemIds
		ordArr, err := s.getUnPaidOrdersByItemIds(itemIds)
		if err != nil {
			log.Error("s.wdb.GetUnPaidOrder", "err", err, "id", urtx.RawId)
			if err == gorm.ErrRecordNotFound {
				log.Warn("need refund about not find order", "id", urtx.RawId)
				// update receipt status is unrefund and waiting refund
				if err = s.wdb.UpdateReceiptStatus(urtx.RawId, schema.UnRefund, nil); err != nil {
					log.Error("s.wdb.UpdateReceiptStatus2", "err", err, "id", urtx.RawId)
				}
			}
			continue
		}

		// check currency, orders currency must == paymentTxSymbol
		if err = checkOrdersCurrency(ordArr, urtx.Symbol); err != nil {
			log.Error("checkOrdersCurrency(ordArr, urtx.Symbol)", "err", err, "urtx", urtx.EverHash)
			if err = s.wdb.UpdateReceiptStatus(urtx.RawId, schema.UnRefund, nil); err != nil {
				log.Error("s.wdb.UpdateReceiptStatus3", "err", err, "id", urtx.RawId)
			}
			continue
		}

		// check amount
		if err = checkOrdersAmount(ordArr, urtx.Amount); err != nil {
			log.Error("checkOrdersAmount(ordArr, urtx.Amount)", "err", err, "urtx", urtx.EverHash)
			if err = s.wdb.UpdateReceiptStatus(urtx.RawId, schema.UnRefund, nil); err != nil {
				log.Error("s.wdb.UpdateReceiptStatus4", "err", err, "id", urtx.RawId)
			}
			continue
		}

		// update order payment status
		dbTx := s.wdb.Db.Begin()
		for _, ord := range ordArr {
			if err = s.wdb.UpdateOrderPay(ord.ID, urtx.EverHash, schema.SuccPayment, dbTx); err != nil {
				log.Error("s.wdb.UpdateOrderPay(ord.ID,schema.SuccPayment,dbTx)", "err", err)
				dbTx.Rollback()
				break
			}
		}

		if err = s.wdb.UpdateReceiptStatus(urtx.RawId, schema.Spent, dbTx); err != nil {
			log.Error("s.wdb.UpdateReceiptStatus(urtx.ID,schema.Spent,dbTx)", "err", err)
			dbTx.Rollback()
			continue
		}
		dbTx.Commit()
	}
}

func checkOrdersAmount(ordArr []schema.Order, txAmount string) error {
	txAmountInt, ok := new(big.Int).SetString(txAmount, 10)
	if !ok {
		return errors.New("txAmount incorrect")
	}
	totalFee := big.NewInt(0)
	for _, ord := range ordArr {
		fee, ok := new(big.Int).SetString(ord.Fee, 10)
		if !ok {
			return errors.New("order fee incorrect")
		}
		totalFee = new(big.Int).Add(totalFee, fee)
	}
	if txAmountInt.Cmp(totalFee) < 0 {
		return errors.New("payAmount fee not enough")
	}
	return nil
}

func checkOrdersCurrency(ordArr []schema.Order, txSymbol string) error {
	for _, ord := range ordArr {
		if strings.ToUpper(ord.Currency) != strings.ToUpper(txSymbol) {
			return errors.New("currency incorrect")
		}
	}
	return nil
}

func parseItemIds(txData string) ([]string, error) {
	itemIds := make([]string, 0)
	res := gjson.Parse(txData)
	// appName must be arseeding
	if res.Get("appName").String() != "arseeding" {
		return nil, errors.New("txData.appName not be arseeding")
	}
	for _, it := range res.Get("itemIds").Array() {
		itemIds = append(itemIds, it.String())
	}
	if len(itemIds) == 0 {
		return nil, errors.New("itemIds is empty")
	}
	return itemIds, nil
}

func (s *Arseeding) getUnPaidOrdersByItemIds(itemIds []string) ([]schema.Order, error) {
	ordArr := make([]schema.Order, 0, len(itemIds))
	for _, itemId := range itemIds {
		ord, err := s.wdb.GetUnPaidOrder(itemId)
		if err != nil {
			log.Error("s.wdb.GetUnPaidOrder(itemId)", "err", err, "itemId", itemId)
			return nil, err
		}
		ordArr = append(ordArr, ord)
	}
	return ordArr, nil
}

func (s *Arseeding) refundReceipt() {
	recpts, err := s.wdb.GetReceiptsByStatus(schema.UnRefund)
	if err != nil {
		log.Error("s.wdb.GetReceiptsByStatus(schema.UnRefund)", "err", err)
		return
	}

	for _, rpt := range recpts {
		// update rpt status is refund
		if err := s.wdb.UpdateReceiptStatus(rpt.RawId, schema.Refund, nil); err != nil {
			log.Error("s.wdb.UpdateReceiptStatus(rpt.ID,schema.Refund,nil)", "err", err, "id", rpt.RawId)
			continue
		}
		// send everTx transfer for refund
		amount, ok := new(big.Int).SetString(rpt.Amount, 10)
		if !ok {
			log.Error("new(big.Int).SetString(rpt.Amount,10) failed", "amt", rpt.Amount)
			continue
		}
		// everTx data
		mmap := map[string]string{
			"appName":        "arseeding",
			"action":         "refund",
			"refundEverHash": rpt.EverHash,
		}
		data, _ := json.Marshal(mmap)
		everTx, err := s.everpaySdk.Transfer(rpt.Symbol, amount, rpt.From, string(data))
		if err != nil { // notice: if refund failed, then need manual check and refund
			log.Error("s.everpaySdk.Transfer", "err", err)
			// update receipt status is unrefund
			if err := s.wdb.UpdateRefundErr(rpt.RawId, err.Error()); err != nil {
				log.Error("s.wdb.UpdateRefundErr(rpt.RawId, err.Error())", "err", err, "id", rpt.RawId)
			}
			continue
		}
		log.Info("refund receipt success...", "receipt everHash", rpt.EverHash, "refund everHash", everTx.HexHash())
	}
}

func (s *Arseeding) onChainBundleItems() {
	ords, err := s.wdb.GetNeedOnChainOrders()
	if err != nil {
		log.Error("s.wdb.GetNeedOnChainOrders()", "err", err)
		return
	}
	// send arTx to arweave
	arTx, onChainItemIds, err := s.onChainOrds(ords)
	if err != nil {
		return
	}

	s.updateOnChainInfo(onChainItemIds, arTx, schema.PendingOnChain)

}

func (s *Arseeding) onChainItemsBySeq() {
	ords, err := s.wdb.GetNeedOnChainOrdersSorted()
	if err != nil {
		log.Error("s.wdb.GetNeedOnChainOrders()", "err", err)
		return
	}

	arTx, onChainItemIds, err := s.onChainOrds(ords)
	if err != nil {
		return
	}

	// block until arTx.confirmation > 3
	status := arTxWatcher(s.arCli, arTx.ID)
	if !status {
		log.Error("watch tx status failed", "ar id", arTx.ID, "status", status)
		return
	}

	s.updateOnChainInfo(onChainItemIds, arTx, schema.SuccOnChain)
}

func (s *Arseeding) onChainOrds(ords []schema.Order) (arTx types.Transaction, onChainItemIds []string, err error) {
	// once total size limit 500 MB
	itemIds := make([]string, 0, len(ords))
	totalSize := int64(0)
	for _, ord := range ords {
		if totalSize >= schema.MaxPerOnChainSize {
			break
		}
		od, exist := s.wdb.ExistProcessedOrderItem(ord.ItemId)
		if exist {
			if err = s.wdb.UpdateOrdOnChainStatus(od.ItemId, od.OnChainStatus, nil); err != nil {
				log.Error("s.wdb.UpdateOrdOnChainStatus(od.ItemId,od.OnChainStatus)", "err", err, "itemId", od.ItemId)
			}
			continue
		}
		itemIds = append(itemIds, ord.ItemId)
		totalSize += ord.Size
	}

	// send arTx to arweave
	return s.onChainBundleTx(itemIds)
}

func (s *Arseeding) updateOnChainInfo(onChainItemIds []string, arTx types.Transaction, onChainStatus string) {
	// insert arTx record
	onChainItemIdsJs, err := json.Marshal(onChainItemIds)
	if err != nil {
		log.Error("json.Marshal(itemIds)", "err", err, "onChainItemIds", onChainItemIds)
		return
	}
	if err = s.wdb.InsertArTx(schema.OnChainTx{
		ArId:      arTx.ID,
		CurHeight: s.cache.GetInfo().Height,
		DataSize:  arTx.DataSize,
		Reward:    arTx.Reward,
		Status:    schema.PendingOnChain,
		ItemIds:   onChainItemIdsJs,
		ItemNum:   len(onChainItemIds),
	}); err != nil {
		log.Error("s.wdb.InsertArTx", "err", err)
		return
	}

	// update order onChainStatus to pending
	for _, itemId := range onChainItemIds {
		if err = s.wdb.UpdateOrdOnChainStatus(itemId, onChainStatus, nil); err != nil {
			log.Error("s.wdb.UpdateOrdOnChainStatus(item.Id,schema.PendingOnChain)", "err", err, "itemId", itemId)
		}
	}
}

func (s *Arseeding) watchArTx() {
	txs, err := s.wdb.GetArTxByStatus(schema.PendingOnChain)
	if err != nil {
		log.Error("s.wdb.GetArTxByStatus(schema.PendingOnChain)", "err", err)
		return
	}

	for _, tx := range txs {
		// check onchain status
		arTxStatus, err := s.arCli.GetTransactionStatus(tx.ArId)
		if err != nil {
			if err != goar.ErrPendingTx && s.cache.GetInfo().Height-tx.CurHeight > 50 {
				// arTx has expired
				if err = s.wdb.UpdateArTxStatus(tx.ArId, schema.FailedOnChain, nil); err != nil {
					log.Error("UpdateArTxStatus(tx.ArId,schema.FailedOnChain)", "err", err)
				}
			}
			continue
		}

		// update status success
		if arTxStatus.NumberOfConfirmations > 3 {
			dbTx := s.wdb.Db.Begin()
			if err = s.wdb.UpdateArTxStatus(tx.ArId, schema.SuccOnChain, dbTx); err != nil {
				log.Error("UpdateArTxStatus(tx.ArId,schema.SuccOnChain)", "err", err)
				dbTx.Rollback()
				continue
			}

			// update order onchain status
			bundleItemIds := make([]string, 0)
			if err = json.Unmarshal(tx.ItemIds, &bundleItemIds); err != nil {
				log.Error("json.Unmarshal(tx.ItemIds,&bundleItemIds)", "err", err, "itemsJs", tx.ItemIds)
				dbTx.Rollback()
				continue
			}

			for _, itemId := range bundleItemIds {
				if err = s.wdb.UpdateOrdOnChainStatus(itemId, schema.SuccOnChain, dbTx); err != nil {
					log.Error("s.wdb.UpdateOrdOnChainStatus(itemId,schema.SuccOnChain,dbTx)", "err", err, "item", itemId)
					dbTx.Rollback()
					continue
				}
			}
			dbTx.Commit()
		}
	}
}

func (s *Arseeding) retryOnChainArTx() {
	txs, err := s.wdb.GetArTxByStatus(schema.FailedOnChain)
	if err != nil {
		log.Error("s.wdb.GetArTxByStatus(schema.PendingOnChain)", "err", err)
		return
	}
	for _, tx := range txs {
		itemIds := make([]string, 0)
		if err = json.Unmarshal(tx.ItemIds, &itemIds); err != nil {
			return
		}
		arTx, _, err := s.onChainBundleTx(itemIds)
		if err != nil {
			return
		}
		// update onChain
		if err = s.wdb.UpdateArTx(tx.ID, arTx.ID, s.cache.GetInfo().Height, arTx.DataSize, arTx.Reward, schema.PendingOnChain); err != nil {
			log.Error("s.wdb.UpdateArTx", "err", err, "id", tx.ID, "arId", arTx.ID)
		}
	}
}

func (s *Arseeding) onChainBundleTx(itemIds []string) (arTx types.Transaction, onChainItemIds []string, err error) {
	onChainItems := make([]types.BundleItem, 0, len(itemIds))
	for _, itemId := range itemIds {
		itemBinary, err := s.store.LoadItemBinary(itemId)
		if err != nil {
			log.Error("s.store.LoadItemBinary(itemId)", "err", err, "itemId", itemId)
			continue
		}
		item, err := utils.DecodeBundleItem(itemBinary)
		if err != nil {
			log.Error("utils.DecodeBundleItem(itemBinary)", "err", err, "itemId", itemId)
			continue
		}
		onChainItems = append(onChainItems, *item)
	}
	if len(onChainItems) == 0 {
		err = errors.New("onChainItems is null")
		return
	}

	// the end off item.Data not be "", because when the case viewblock decode failed. // todo viewblock used stream function decode item, so this is a bug for them
	endItem := onChainItems[len(onChainItems)-1]
	if endItem.Data == "" {
		// find a data != "" item and push to end off
		idx := -1
		for i, item := range onChainItems {
			if item.Data != "" {
				idx = i
				break
			}
		}
		if idx == -1 {
			err = errors.New("all bundle items data are null")
			return
		}
		newEndItem := onChainItems[idx]
		onChainItems = append(onChainItems[:idx], onChainItems[idx+1:]...)
		onChainItems = append(onChainItems, newEndItem)
	}

	// get onChainItemIds
	for _, item := range onChainItems {
		onChainItemIds = append(onChainItemIds, item.Id)
	}

	// assemble and send to arweave
	bundle, err := utils.NewBundle(onChainItems...)
	if err != nil {
		log.Error("utils.NewBundle(onChainItems...)", "err", err)
		return
	}

	// verify bundle, ensure that the bundle is exactly right before sending
	if _, err = utils.DecodeBundle(bundle.BundleBinary); err != nil {
		err = errors.New(fmt.Sprintf("Verify bundle failed; err:%v", err))
		return
	}

	arTxtags := []types.Tag{
		{Name: "App-Name", Value: "arseeding"},
		{Name: "App-Version", Value: "1.0.0"},
		{Name: "Action", Value: "Bundle"},
		{Name: "Protocol-Name", Value: "BAR"},
		{Name: "Action", Value: "Burn"},
		{Name: "App-Name", Value: "SmartWeaveAction"},
		{Name: "App-Version", Value: "0.3.0"},
		{Name: "Input", Value: `{"function":"mint"}`},
		{Name: "Contract", Value: "VFr3Bk-uM-motpNNkkFg4lNW1BMmSfzqsVO551Ho4hA"},
	}

	if len(s.customTags) > 0 {
		arTxtags = append(s.customTags, arTxtags...)
	}

	// speed arTx Fee
	price := calculatePrice(s.cache.GetFee(), int64(len(bundle.BundleBinary)))
	speedFactor := calculateFactor(price, s.config.GetSpeedFee())
	arTx, err = s.bundler.SendBundleTxSpeedUp(bundle.BundleBinary, arTxtags, speedFactor)
	if err != nil {
		log.Error("s.bundler.SendBundleTxSpeedUp(bundle.BundleBinary,arTxtags)", "err", err)
		return
	}
	log.Info("send bundle arTx", "arTx", arTx.ID)

	// arseeding broadcast tx data
	if err := s.arseedCli.SubmitTx(arTx); err != nil {
		log.Error("s.arseedCli.SubmitTx(arTx)", "err", err, "arId", arTx.ID)
	}
	return
}

func (s *Arseeding) processExpiredOrd() {
	ords, err := s.wdb.GetExpiredOrders()
	if err != nil {
		log.Error("GetExpiredOrders()", "err", err)
		return
	}
	for _, ord := range ords {
		if err = s.wdb.UpdateOrdToExpiredStatus(ord.ID); err != nil {
			log.Error("UpdateOrdToExpiredStatus", "err", err, "id", ord.ID)
			continue
		}
		// can not delete
		// 1. exist paid order
		if s.wdb.ExistPaidOrd(ord.ItemId) {
			continue
		}
		// 2. this order not the latest unpaid order
		if !s.wdb.IsLatestUnpaidOrd(ord.ItemId, ord.PaymentExpiredTime) {
			continue
		}
		// delete bundle item from store
		if err = s.DelItem(ord.ItemId); err != nil {
			log.Error("DelItem", "err", err, "itemId", ord.ItemId)
			continue
		}
		// delete manifest table
		if err = s.wdb.DelManifest(ord.ItemId); err != nil {
			log.Error("s.wdb.DelManifest", "err", err, "itemId", ord.ItemId)
			continue
		}
	}
}

func (s *Arseeding) parseAndSaveBundleTx() {
	arIds, err := s.store.LoadWaitParseBundleArIds()
	if err != nil {
		if err != schema.ErrNotExist {
			log.Error("s.store.LoadWaitParseBundleArIds()", "err", err)
		}
		return
	}
	for _, arId := range arIds {
		// get tx data
		arTxMeta, err := s.store.LoadTxMeta(arId)
		if err != nil {
			log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
			continue
		}

		data, err := getArTxData(arTxMeta.DataRoot, arTxMeta.DataSize, s.store)
		if err != nil {
			log.Error("get data failed, if is not_exist_record,then wait submit chunks fully", "err", err, "arId", arId)
			continue
		}
		if err := s.ParseAndSaveBundleItems(arId, data); err != nil {
			log.Error("ParseAndSaveBundleItems", "err", err, "arId", arId)
		}
		// del wait db
		if err = s.store.DelParsedBundleArId(arId); err != nil {
			log.Error("DelParsedBundleArId", "err", err, "arId", arId)
		}
	}
}

func (s *Arseeding) updateBundler() {
	// update bundler balance
	addr := s.bundler.Signer.Address
	bal, err := s.arCli.GetWalletBalance(addr)
	if err != nil {
		return
	}
	metricBundlerBalance(bal, addr)
}

func filterPeers(peers []string, constTx *types.Transaction) map[string]bool {
	var mu sync.Mutex
	var wg sync.WaitGroup
	availablePeers := make(map[string]bool, 0)
	p, err := ants.NewPoolWithFunc(50, func(peer interface{}) {
		defer wg.Done()
		pStr := peer.(string)
		pNode := goar.NewTempConn()
		pNode.SetTempConnUrl("http://" + pStr)
		pNode.SetTimeout(time.Second * 10)
		_, code, err := pNode.SubmitTransaction(constTx)
		if err != nil {
			return
		}
		// if the resp like this ,we believe this peer is available
		if code/100 == 2 || code/100 == 4 {
			mu.Lock()
			availablePeers[pStr] = true
			mu.Unlock()
		}
	})
	// submit a legal Tx to peers, if the response is timely and acceptable, then the peer is good for submit tx
	for _, peer := range peers {
		wg.Add(1)
		err = p.Invoke(peer)
		if err != nil {
			log.Warn("concurrency err", "err", err)
		}
	}
	wg.Wait()
	p.Release()
	return availablePeers
}

func updatePeerMap(oldPeerMap map[string]int64, availablePeers map[string]bool) map[string]int64 {
	for peer, cnt := range oldPeerMap {
		if _, ok := availablePeers[peer]; !ok {
			if cnt > 0 {
				oldPeerMap[peer] -= 1
			}
		} else {
			oldPeerMap[peer] += 1
		}
	}

	for peer, _ := range availablePeers {
		if _, ok := oldPeerMap[peer]; !ok {
			oldPeerMap[peer] = 1
		}
	}
	return oldPeerMap
}

func calculateFactor(price, speedFee int64) int64 {
	if speedFee == 0 {
		return 0
	}
	val := (price+speedFee)*100/price - 100
	if val == 0 {
		val = 1
	}
	return val
}

func arTxWatcher(arCli *goar.Client, arTxHash string) bool {
	// loop watcher on chain tx status
	// total time 59 minute
	tmp := 0
	for i := 1; i <= 21; i++ {
		// sleep
		num := 60 + i*10
		time.Sleep(time.Second * time.Duration(num))

		tmp += num
		log.Debug("watcher tx sleep time", "wait total time(s)", tmp)
		log.Debug("retry get tx status", "txHash", arTxHash)

		// watcher on-chain tx
		status, err := arCli.GetTransactionStatus(arTxHash)
		if err != nil {
			if err.Error() == goar.ErrPendingTx.Error() {
				log.Debug("tx is pending", "txHash", arTxHash)
			} else {
				log.Error("get tx status", "err", err, "txHash", arTxHash)
			}
			continue
		}

		// when err is nil
		// confirms block height must >= 3
		if status.NumberOfConfirmations < 3 {
			log.Debug("arseeding send sequence tx must more than 2 block confirms", "txHash", arTxHash, "currentConfirms", status.NumberOfConfirmations)
			continue
		} else {
			return true
		}
	}
	return false
}
