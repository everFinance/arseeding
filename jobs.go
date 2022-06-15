package arseeding

import (
	"encoding/json"
	"errors"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/everpay/account"
	"github.com/everFinance/everpay/config"
	sdkSchema "github.com/everFinance/everpay/sdk/schema"
	paySchema "github.com/everFinance/everpay/token/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"gorm.io/gorm"
	"math/big"
	"strings"
	"time"
)

func (s *Arseeding) runJobs() {
	// update cache
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.updateAnchor)
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.updateArFee)
	s.scheduler.Every(30).Seconds().SingletonMode().Do(s.updateInfo)
	s.scheduler.Every(30).Minute().SingletonMode().Do(s.updatePeerList)

	// about bundle
	s.scheduler.Every(5).Minute().SingletonMode().Do(s.updateTokenPrice)
	s.scheduler.Every(1).Minute().SingletonMode().Do(s.updateBundlePerFee)
	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.mergeReceiptAndOrder)
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.refundReceipt)
	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.onChainBundleItems)
	s.scheduler.Every(5).Minute().SingletonMode().Do(s.watchArTx)
	s.scheduler.Every(5).Minute().SingletonMode().Do(s.retryOnChainArTx)
	go s.watchEverReceiptTxs()
	s.scheduler.Every(1).Minute().SingletonMode().Do(s.processExpiredOrd)
	s.scheduler.Every(1).Minute().SingletonMode().Do(s.parseAndSaveBundleTx)

	// manager jobStatus
	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.watcherAndCloseJobs)

	s.scheduler.StartAsync()
}

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
		price, err := config.GetTokenPriceByRedstone(tp.Symbol, "AR")
		if err != nil {
			log.Error("config.GetTokenPriceByRedstone(tp.Symbol,\"AR\")", "err", err, "symbol", tp.Symbol)
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

func (s *Arseeding) watcherAndCloseJobs() {
	jobs := s.taskMg.GetTasks()
	now := time.Now().Unix()
	for _, job := range jobs {
		if job.Close || job.Timestamp == 0 { // timestamp == 0  means do not start
			continue
		}
		// spend time not more than 30 minutes
		if now-job.Timestamp > 30*60 {
			if err := s.taskMg.CloseTask(job.ArId, job.TaskType); err != nil {
				log.Error("watcherAndCloseJobs closeJob", "err", err, "jobId", assembleTaskId(job.ArId, job.TaskType))
				continue
			}
		}
	}
}

func (s *Arseeding) updateAnchor() {
	anchor, err := fetchAnchor(s.arCli, s.cache.GetPeers())
	if err == nil {
		s.cache.UpdateAnchor(anchor)
	}
}

// update arweave network info

func (s *Arseeding) updateInfo() {
	info, err := fetchArInfo(s.arCli, s.cache.GetPeers())
	if err == nil {
		s.cache.UpdateInfo(info)
	}
}

func (s *Arseeding) updateArFee() {
	txPrice, err := fetchArFee(s.arCli, s.cache.GetPeers())
	if err == nil {
		s.cache.UpdateFee(txPrice)
	}
}

// update peer list, check peer available, store in db
// TODO update peerList concurrency
func (s *Arseeding) updatePeerList() {
	visPeer := make(map[string]bool, 0) // record already handled peer
	updatedPeers := make([]string, 0)
	pNode := goar.NewTempConn()
	for _, peer := range s.cache.GetPeers() {
		pNode.SetTempConnUrl("http://" + peer)
		newPeers, err := pNode.GetPeers()
		if err != nil {
			log.Warn("bad peer")
			continue
		}
		for _, newPeer := range newPeers {
			if _, ok := visPeer[newPeer]; ok {
				continue
			}
			if checkAvailable(newPeer) {
				updatedPeers = append(updatedPeers, newPeer)
			}
			visPeer[newPeer] = true
		}
	}

	s.cache.UpdatePeers(updatedPeers)
	err := s.store.SavePeers(updatedPeers)
	if err != nil {
		log.Warn("save new peer list fail")
	}
}

// check the peer is available and health
// return true temporary
// TODO
func checkAvailable(peer string) bool {
	return true
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

	for _, urt := range unspentRpts {
		signer := urt.From
		ord, err := s.wdb.GetUnPaidOrder(signer, urt.Symbol, urt.Amount)
		if err != nil {
			log.Error("s.wdb.GetSignerOrder", "err", err, "signer", signer, "symbol", urt.Symbol, "fee", urt.Amount)
			if err == gorm.ErrRecordNotFound {
				// update receipt status is unrefund and waiting refund
				if err = s.wdb.UpdateReceiptStatus(urt.RawId, schema.UnRefund, nil); err != nil {
					log.Error("s.wdb.UpdateReceiptStatus", "err", err, "id", urt.RawId)
				}
			}
			continue
		}

		// update order payment status
		dbTx := s.wdb.Db.Begin()
		if err = s.wdb.UpdateOrderPay(ord.ID, urt.EverHash, schema.SuccPayment, dbTx); err != nil {
			log.Error("s.wdb.UpdateOrderPay(ord.ID,schema.SuccPayment,dbTx)", "err", err)
			dbTx.Rollback()
			continue
		}
		if err = s.wdb.UpdateReceiptStatus(urt.RawId, schema.Spent, dbTx); err != nil {
			log.Error("s.wdb.UpdateReceiptStatus(urt.ID,schema.Spent,dbTx)", "err", err)
			dbTx.Rollback()
			continue
		}
		dbTx.Commit()
	}
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
		everTx, err := s.everpaySdk.Transfer(rpt.Symbol, amount, rpt.From, rpt.EverHash)
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
	// once total size limit 500 MB
	itemIds := make([]string, 0, len(ords))
	totalSize := int64(0)
	for _, ord := range ords {
		if totalSize >= schema.MaxPerOnChainSize {
			break
		}
		itemIds = append(itemIds, ord.ItemId)
		totalSize += ord.Size
	}

	arTx, onChainItemIds, err := s.onChainBundleTx(itemIds)
	if err != nil {
		return
	}

	// insert arTx record
	onChainItemIdsJs, err := json.Marshal(onChainItemIds)
	if err != nil {
		log.Error("json.Marshal(itemIds)", "err", err, "onChainItemIds", onChainItemIds)
		return
	}
	if err = s.wdb.InsertArTx(schema.OnChainTx{
		ArId:      arTx.ID,
		CurHeight: s.arInfo.Height,
		Status:    schema.PendingOnChain,
		ItemIds:   onChainItemIdsJs,
	}); err != nil {
		log.Error("s.wdb.InsertArTx", "err", err)
		return
	}

	// update order onChainStatus to pending
	for _, itemId := range onChainItemIds {
		if err = s.wdb.UpdateOrdOnChainStatus(itemId, schema.PendingOnChain, nil); err != nil {
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
		if s.arInfo.Height-tx.CurHeight > 50 {
			// arTx has expired
			if err = s.wdb.UpdateArTxStatus(tx.ArId, schema.FailedOnChain, nil); err != nil {
				log.Error("UpdateArTxStatus(tx.ArId,schema.FailedOnChain)", "err", err)
			}
			continue
		}

		// check onchain status
		arTxStatus, err := s.arCli.GetTransactionStatus(tx.ArId)
		if err != nil {
			log.Error("s.arCli.GetTransactionStatus(tx.ArId)", "err", err, "arId", tx.ArId)
			continue
		}

		// update status success
		if arTxStatus.NumberOfConfirmations > 3 {
			dbTx := s.wdb.Db.Begin()
			if err = s.wdb.UpdateArTxStatus(tx.ArId, schema.SuccOnChain, dbTx); err != nil {
				log.Error("UpdateArTxStatus(tx.ArId,schema.SuccOnChain)", "err", err)
				dbTx.Rollback()
			}

			// update order onchain status
			bundleItemIds := make([]string, 0)
			if err = json.Unmarshal(tx.ItemIds, &bundleItemIds); err != nil {
				log.Error("json.Unmarshal(tx.ItemIds,&bundleItemIds)", "err", err, "itemsJs", tx.ItemIds)
				dbTx.Rollback()
				return
			}

			for _, itemId := range bundleItemIds {
				if err = s.wdb.UpdateOrdOnChainStatus(itemId, schema.SuccOnChain, dbTx); err != nil {
					log.Error("s.wdb.UpdateOrdOnChainStatus(itemId,schema.SuccOnChain,dbTx)", "err", err, "item", itemId)
					dbTx.Rollback()
					return
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
		if err = s.wdb.UpdateArTx(tx.ID, arTx.ID, s.arInfo.Height, schema.PendingOnChain); err != nil {
			log.Error("s.wdb.UpdateArTx", "err", err, "id", tx.ID, "arId", arTx.ID)
		}
	}
}

func (s *Arseeding) onChainBundleTx(itemIds []string) (arTx types.Transaction, onChainItemIds []string, err error) {
	onChainItems := make([]types.BundleItem, 0, len(itemIds))
	onChainItemIds = make([]string, 0, len(itemIds))
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
		onChainItemIds = append(onChainItemIds, item.Id)
	}
	if len(onChainItems) == 0 {
		err = errors.New("onChainItems is null")
		return
	}

	// assemble and send to arweave
	bundle, err := utils.NewBundle(onChainItems...)
	if err != nil {
		log.Error("utils.NewBundle(onChainItems...)", "err", err)
		return
	}
	arTxtags := []types.Tag{
		{Name: "Application", Value: "arseeding"},
		{Name: "Action", Value: "Bundle"},
	}
	arTx, err = s.bundler.SendBundleTxSpeedUp(bundle.BundleBinary, arTxtags, 20)
	if err != nil {
		log.Error("s.bundler.SendBundleTxSpeedUp(bundle.BundleBinary,arTxtags,20)", "err", err)
		return
	}
	log.Info("send bundle arTx", "arTx", arTx.ID)

	// submit to arseeding
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
		// delete bundle item from store
		if err = s.DelItem(ord.ItemId); err != nil {
			log.Error("DelItem", "err", err, "itemId", ord.ItemId)
		}
	}
}

func (s *Arseeding) parseAndSaveBundleTx() {
	arIds, err := s.store.LoadWaitParseBundleArIds()
	if err != nil {
		log.Error("s.store.LoadWaitParseBundleArIds()", "err", err)
		return
	}
	for _, arId := range arIds {
		// get tx data
		arTxMeta, err := s.store.LoadTxMeta(arId)
		if err != nil {
			log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
			continue
		}

		data, err := getData(arTxMeta.DataRoot, arTxMeta.DataSize, s.store)
		if err != nil {
			log.Error("get data failed, if is not_exist_record,then wait submit chunks fully", "err", err, "arId", arId)
			continue
		}
		if err := s.ParseAndSaveBundleItems(arId, data); err != nil {
			log.Error("ParseAndSaveBundleItems", "err", err, "arId", arId)
			continue
		}
		// del wait db
		if err = s.store.DelParsedBundleArId(arId); err != nil {
			log.Error("DelParsedBundleArId", "err", err, "arId", arId)
		}
	}
}
