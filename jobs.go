package arseeding

import (
	"encoding/json"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/everpay/account"
	"github.com/everFinance/everpay/config"
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
	s.scheduler.Every(3).Seconds().SingletonMode().Do(s.watcherReceiptTxs)
	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.mergeReceiptAndOrder)
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.refundReceipt)
	s.scheduler.Every(30).Seconds().SingletonMode().Do(s.onChainBundleItems)
	s.scheduler.Every(5).Minute().SingletonMode().Do(s.watcherOnChainTx)

	// manager jobStatus
	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.watcherAndCloseJobs)

	s.scheduler.StartAsync()
}

func (s *Arseeding) updateTokenPrice() {
	// update symbol
	tps := make([]schema.TokenPrice, 0)
	for _, tok := range s.paySdk.GetTokens() {
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
	jobs := s.jobManager.GetJobs()
	now := time.Now().Unix()
	for _, job := range jobs {
		if job.Close || job.Timestamp == 0 { // timestamp == 0  means do not start
			continue
		}
		// spend time not more than 30 minutes
		if now-job.Timestamp > 30*60 {
			if err := s.jobManager.CloseJob(job.ArId, job.JobType); err != nil {
				log.Error("watcherAndCloseJobs closeJob", "err", err, "jobId", assembleId(job.ArId, job.JobType))
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

func (s *Arseeding) watcherReceiptTxs() {
	startPage, err := s.wdb.GetLastPage()
	if err != nil {
		log.Error("s.wdb.GetLastPage()", "err", err)
	}
	page := startPage // todo use new everpay/sdk function
	for {
		txs, err := s.paySdk.Cli.TxsByAcc(s.bundler.Signer.Address, page, "asc", "", paySchema.TxActionTransfer, "")
		if err != nil {
			log.Error("pay.client.TxsByAcc failed", "err", err)
			return
		}
		records := make([]schema.ReceiptEverTx, 0, len(txs.Txs.Txs))

		for _, tt := range txs.Txs.Txs {
			if tt.To != s.bundler.Signer.Address {
				continue
			}
			_, from, err := account.IDCheck(tt.From)
			if err != nil {
				log.Error("account.IDCheck(tt.From)", "err", err, "from", tt.From)
				continue
			}
			records = append(records, schema.ReceiptEverTx{
				EverHash: tt.EverHash,
				Nonce:    tt.Nonce,
				Symbol:   tt.TokenSymbol,
				Action:   tt.Action,
				From:     from,
				Amount:   tt.Amount,
				Data:     tt.Data,
				Page:     txs.Txs.CurrentPage,
				Status:   schema.UnSpent,
			})
		}
		if err := s.wdb.InsertReceiptTx(records); err != nil {
			log.Error("s.wdb.InsertReceiptTx(records)", "err", err)
			return
		}

		if txs.Txs.CurrentPage == txs.Txs.TotalPages {
			return
		}

		page++
	}
}

func (s *Arseeding) mergeReceiptAndOrder() {
	unspentReceipts, err := s.wdb.GetReceiptsByStatus(schema.UnSpent)
	if err != nil {
		log.Error("s.wdb.GetUnSpentReceipts()", "err", err)
		return
	}

	for _, urt := range unspentReceipts {
		signer := urt.From
		ord, err := s.wdb.GetUnPaidOrder(signer, urt.Symbol, urt.Amount)
		if err != nil {
			log.Error("s.wdb.GetSignerOrder", "err", err, "signer", signer, "symbol", urt.Symbol, "fee", urt.Amount)
			if err == gorm.ErrRecordNotFound {
				// update receipt status is unrefund
				if err = s.wdb.UpdateReceiptStatus(urt.ID, schema.UnRefund, nil); err != nil {
					log.Error("s.wdb.UpdateReceiptStatus", "err", err, "id", urt.ID)
				}
			}
			continue
		}

		// check order time expired
		everTxNonce := urt.Nonce
		if ord.PaymentExpiredTime*1000 < everTxNonce {
			// expired
			dbTx := s.wdb.Db.Begin()
			if err = s.wdb.UpdateOrderPay(ord.ID, urt.EverHash, schema.ExpiredPayment, dbTx); err != nil {
				log.Error("s.wdb.UpdateOrderPay(ord.ID,schema.ExpiredPayment,dbTx)", "err", err)
				dbTx.Rollback()
				continue
			}

			if err = s.wdb.UpdateReceiptStatus(urt.ID, schema.UnRefund, dbTx); err != nil {
				log.Error("s.wdb.UpdateReceiptStatus(urt.ID,schema.UnRefund,dbTx)", "err", err)
				dbTx.Rollback()
				continue
			}
			dbTx.Commit()
		}

		// update order payment status
		dbTx := s.wdb.Db.Begin()
		if err = s.wdb.UpdateOrderPay(ord.ID, urt.EverHash, schema.SuccPayment, dbTx); err != nil {
			log.Error("s.wdb.UpdateOrderPay(ord.ID,schema.SuccPayment,dbTx)", "err", err)
			dbTx.Rollback()
			continue
		}
		if err = s.wdb.UpdateReceiptStatus(urt.ID, schema.Spent, dbTx); err != nil {
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
		if err := s.wdb.UpdateReceiptStatus(rpt.ID, schema.Refund, nil); err != nil {
			log.Error("s.wdb.UpdateReceiptStatus(rpt.ID,schema.Refund,nil)", "err", err, "id", rpt.ID)
			continue
		}
		// send everTx transfer for refund
		amount, ok := new(big.Int).SetString(rpt.Amount, 10)
		if !ok {
			log.Error("new(big.Int).SetString(rpt.Amount,10) failed", "amt", rpt.Amount)
			continue
		}
		everTx, err := s.paySdk.Transfer(rpt.Symbol, amount, rpt.From, rpt.EverHash)
		if err != nil {
			log.Error("s.paySdk.Transfer", "err", err)
			// update receipt status is unrefund
			if err := s.wdb.UpdateReceiptStatus(rpt.ID, schema.UnRefund, nil); err != nil {
				log.Error("s.wdb.UpdateReceiptStatus(rpt.ID,schema.UnRefund,nil)", "err", err, "id", rpt.ID)
			}
			continue
		}
		log.Info("refund receipt success", "receipt everHash", rpt.EverHash, "refund everHash", everTx.HexHash())
	}
}

func (s *Arseeding) onChainBundleItems() {
	ords, err := s.wdb.GetNeedOnChainOrders()
	if err != nil {
		log.Error("s.wdb.GetNeedOnChainOrders()", "err", err)
		return
	}
	// once total size limit 500 MB
	filterItemIds := make([]string, 0, len(ords))
	totalSize := int64(0)
	for _, ord := range ords {
		if totalSize >= schema.MaxPerOnChainSize {
			break
		}
		filterItemIds = append(filterItemIds, ord.ItemId)
		totalSize += ord.Size
	}

	// assemble Bundle items
	items := make([]types.BundleItem, 0, len(filterItemIds))
	for _, id := range filterItemIds {
		itemBinary, err := s.store.LoadItemBinary(id)
		if err != nil {
			log.Error("s.store.LoadItemBinary(id)", "err", err, "id", id)
			continue
		}
		item, err := utils.DecodeBundleItem(itemBinary)
		if err != nil {
			log.Error("utils.DecodeBundleItem(itemBinary)", "err", err, "id", id)
			continue
		}
		items = append(items, *item)
	}
	if len(items) == 0 {
		return
	}
	// assemble bundle
	bundle, err := utils.NewBundle(items...)
	if err != nil {
		log.Error("utils.NewBundle(items...)", "err", err)
		return
	}
	arTxtags := []types.Tag{
		{Name: "Application", Value: "arseeding"},
		{Name: "Action", Value: "Bundle"},
	}
	arTx, err := s.bundler.SendBundleTxSpeedUp(bundle.BundleBinary, arTxtags, 20)
	if err != nil {
		log.Error("s.bundler.SendBundleTxSpeedUp(bundle.BundleBinary,arTxtags,20)", "err", err)
		return
	}
	log.Info("send bundle arTx", "arTx", arTx.ID)

	// insert arTx record
	itemIdsJs, err := json.Marshal(filterItemIds)
	if err != nil {
		log.Error("json.Marshal(filterItemIds)", "err", err, "ids", filterItemIds)
		return
	}
	if err = s.wdb.InsertOnChainTx(schema.OnChainTx{
		ArId:      arTx.ID,
		CurHeight: s.arInfo.Height,
		Status:    schema.PendingOnChain,
		ItemIds:   itemIdsJs,
	}); err != nil {
		log.Error("s.wdb.InsertOnChainTx", "err", err)
		return
	}

	// update order onChainStatus to pending
	for _, item := range items {
		if err := s.wdb.UpdateOnChainStatus(item.Id, schema.PendingOnChain); err != nil {
			log.Error("s.wdb.UpdateOnChainStatus(item.Id,schema.PendingOnChain)", "err", err, "itemId", item.Id)
		}
	}
}

func (s *Arseeding) watcherOnChainTx() {
	txs, err := s.wdb.GetPendingOnChainTx()
	if err != nil {
		log.Error("s.wdb.GetPendingOnChainTx()", "err", err)
		return
	}

	for _, tx := range txs {
		if s.arInfo.Height-tx.CurHeight > 50 {
			// arTx has expired
			if err = s.wdb.UpdateOnChainTxStatus(tx.ArId, schema.FailedOnChain); err != nil {
				log.Error("UpdateOnChainTxStatus(tx.ArId,schema.FailedOnChain)", "err", err)
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
			if err = s.wdb.UpdateOnChainTxStatus(tx.ArId, schema.SuccOnChain); err != nil {
				log.Error("UpdateOnChainTxStatus(tx.ArId,schema.SuccOnChain)", "err", err)
			}
			// todo update order onChain status
		}
	}
}
