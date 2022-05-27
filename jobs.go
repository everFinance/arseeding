package arseeding

import (
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/everpay/config"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/panjf2000/ants/v2"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"
	"sync"
	"time"
)

func (s *Arseeding) runJobs() {
	s.scheduler.Every(1).Minute().SingletonMode().Do(s.updatePeers)
	s.scheduler.Every(5).Minute().SingletonMode().Do(s.updateTokenPrice)
	s.scheduler.Every(1).Minute().SingletonMode().Do(s.updateBundlePerFee)
	s.scheduler.Every(5).Minute().SingletonMode().Do(s.updateArFee)

	s.scheduler.Every(2).Seconds().SingletonMode().Do(s.runBroadcastJobs)
	s.scheduler.Every(2).Seconds().SingletonMode().Do(s.runSyncJobs)

	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.watcherAndCloseJobs)

	s.scheduler.StartAsync()
}

func (s *Arseeding) updatePeers() {
	peers, err := s.arCli.GetPeers()
	if err != nil {
		return
	}
	if len(peers) == 0 {
		return
	}
	// update
	s.peers = peers
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

	// update price
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
			log.Error("s.wdb.UpdatePrice(tp.Symbol,price)", "err", err, "symbol", tp.Symbol, "price", price)
		}
	}
}

func (s *Arseeding) updateArFee() {
	chunk0 := make([]byte, 0)
	chunk1 := make([]byte, 1)
	reword0, err := s.arCli.GetTransactionPrice(chunk0, nil)
	if err != nil {
		log.Error("s.arCli.GetTransactionPrice(chunk0, nil)", "err", err)
		return
	}
	reword1, err := s.arCli.GetTransactionPrice(chunk1, nil)
	if err != nil {
		log.Error("s.arCli.GetTransactionPrice(chunk1, nil)", "err", err)
		return
	}
	baseFee := reword0
	perChunkFee := reword1 - reword0
	if perChunkFee <= 0 {
		log.Error("perChunkFee incorrect", "reword0", reword0, "reword1", reword1)
		return
	}

	if err = s.wdb.UpdateArFee(baseFee, perChunkFee); err != nil {
		log.Error("s.wdb.UpdateArFee(baseFee,perChunkFee)", "err", err, "baseFee", baseFee, "perChunkFee", perChunkFee)
	}
}

func (s *Arseeding) updateBundlePerFee() {
	feeMap, err := s.getBundlePerFees()
	if err != nil {
		log.Error("s.getBundlePerFees()", "err", err)
		return
	}
	s.bundlePerFeeMap = feeMap
}

func (s *Arseeding) getBundlePerFees() (map[string]schema.Fee, error) {
	tps, err := s.wdb.GetPrices()
	if err != nil {
		return nil, err
	}

	arFee, err := s.wdb.GetArFee()
	if err != nil {
		return nil, err
	}

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

func (s *Arseeding) runBroadcastJobs() {
	arIds, err := s.store.LoadPendingPool(jobTypeBroadcast, 50)
	if err != nil {
		log.Error("s.store.LoadPendingPool(jobTypeBroadcast, 20)", "err", err)
		return
	}
	if len(arIds) == 0 {
		return
	}

	log.Debug("load jobTypeBroadcast pending pool", "number", len(arIds))
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(50, func(i interface{}) {
		defer wg.Done()
		arId := i.(string)
		if err := s.processBroadcastJob(arId); err != nil {
			log.Error("processBroadcastJob", "err", err, "arId", arId)
			return
		}
	})
	defer p.Release()

	for _, arId := range arIds {
		wg.Add(1)
		_ = p.Invoke(arId)
	}
	wg.Wait()

	if err := s.setProcessedJobs(arIds, jobTypeBroadcast); err != nil {
		log.Error("s.setProcessedJobs(arIds,jobTypeBroadcast)", "err", err)
	} else {
		log.Debug("run broadcast jobs success", "broadcastJobs number", len(arIds))
	}
}

func (s *Arseeding) runSyncJobs() {
	arIds, err := s.store.LoadPendingPool(jobTypeSync, 100)
	if err != nil {
		log.Error("s.store.LoadPendingPool(jobTypeSync, 50)", "err", err)
		return
	}

	if len(arIds) == 0 {
		return
	}

	log.Debug("load jobTypeSync pending pool", "number", len(arIds))
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(100, func(i interface{}) {
		defer wg.Done()
		arId := i.(string)
		if err := s.processSyncJob(arId); err != nil {
			log.Error("processSyncJob", "err", err, "arId", arId)
			return
		}
	})
	defer p.Release()

	for _, arId := range arIds {
		wg.Add(1)
		_ = p.Invoke(arId)
	}
	wg.Wait()

	if err := s.setProcessedJobs(arIds, jobTypeSync); err != nil {
		log.Error("s.setProcessedJobs(arIds, jobTypeSync)", "err", err)
	} else {
		log.Debug("run sync jobs success", "syncJobs number", len(arIds))
	}
}

func (s *Arseeding) processBroadcastJob(arId string) (err error) {
	// job manager set
	if s.jobManager.IsClosed(arId, jobTypeBroadcast) {
		log.Warn("broadcast job was closed", "arId", arId)
		return
	}
	if err = s.jobManager.JobBeginSet(arId, jobTypeBroadcast, len(s.peers)); err != nil {
		log.Error("s.jobManager.JobBeginSet(arId, jobTypeBroadcast)", "err", err, "arId", arId)
		return
	}

	if !s.store.IsExistTxMeta(arId) {
		return ErrNotExist
	}
	txMeta, err := s.store.LoadTxMeta(arId)
	if err != nil {
		log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
		return err
	}
	txData, err := getData(txMeta.DataRoot, txMeta.DataSize, s.store)
	if err != nil {
		log.Error("getData(txMeta.DataRoot,txMeta.DataSize,s.store)", "err", err, "arId", arId)
		return err
	}

	txMetaPosted := true
	// check this tx whether on chain
	_, err = s.arCli.GetTransactionStatus(arId)
	if err == goar.ErrPendingTx || err == goar.ErrNotFound {
		txMetaPosted = false
		err = nil
	}
	// generate tx chunks
	utils.PrepareChunks(txMeta, txData)
	txMeta.Data = utils.Base64Encode(txData)

	s.jobManager.BroadcastData(arId, jobTypeBroadcast, txMeta, s.peers, txMetaPosted)
	return
}

func (s *Arseeding) processSyncJob(arId string) (err error) {
	// 0. job manager set
	if s.jobManager.IsClosed(arId, jobTypeSync) {
		return
	}
	if err = s.jobManager.JobBeginSet(arId, jobTypeSync, len(s.peers)); err != nil {
		log.Error("s.jobManager.JobBeginSet(arId, jobTypeSync)", "err", err, "arId", arId)
		return
	}

	// 1. sync arTxMeta
	arTxMeta := &types.Transaction{}
	arTxMeta, err = s.store.LoadTxMeta(arId)
	if err != nil {
		// get txMeta from arweave network
		arTxMeta, err = s.arCli.GetUnconfirmedTx(arId) // this api can return all tx (unconfirmed and confirmed)
		if err != nil {
			// get tx from peers
			arTxMeta, err = s.jobManager.GetUnconfirmedTxFromPeers(arId, jobTypeSync, s.peers)
			if err != nil {
				return err
			}
		}
	}
	if len(arTxMeta.ID) == 0 { // arTxMeta can not be null
		return fmt.Errorf("get arTxMeta failed; arId: %s", arId)
	}

	// if arTxMeta not exist local, store it
	if !s.store.IsExistTxMeta(arId) {
		// store txMeta
		if err := s.store.SaveTxMeta(*arTxMeta); err != nil {
			log.Error("s.store.SaveTxMeta(arTx)", "err", err, "arTx", arTxMeta.ID)
			return err
		}
		// store txDataEndOffset
		if err := s.syncAddTxDataEndOffset(arTxMeta.DataRoot, arTxMeta.DataSize); err != nil {
			log.Error("s.syncAddTxDataEndOffset(arTxMeta.DataRoot,arTxMeta.DataSize)", "err", err, "arId", arId)
			return err
		}
	}

	// 2. sync data
	size, ok := new(big.Int).SetString(arTxMeta.DataSize, 10)
	if !ok {
		return fmt.Errorf("new(big.Int).SetString(arTxMeta.DataSize,10) failed; dataSize: %s", arTxMeta.DataSize)
	}

	if size.Cmp(big.NewInt(0)) <= 0 {
		// not need to sync tx data, so this job is success
		s.jobManager.IncSuccessed(arId, jobTypeSync)
		return nil
	}

	// get data
	var data []byte
	data, err = getData(arTxMeta.DataRoot, arTxMeta.DataSize, s.store) // get data from local
	if err == nil {
		return nil // local exist data
	} else {
		// need get tx data from arweave network
		data, err = s.arCli.GetTransactionDataByGateway(arId)
		if err != nil {
			data, err = s.jobManager.GetTxDataFromPeers(arId, jobTypeSync, s.peers)
			if err != nil {
				log.Error("get data failed", "err", err, "arId", arId)
				return err
			}
		} else {
			s.jobManager.IncSuccessed(arId, jobTypeSync)
		}
	}
	// store data to local
	return setTxDataChunks(*arTxMeta, data, s.store)
}

func (s *Arseeding) setProcessedJobs(arIds []string, jobType string) error {
	// process job status
	for _, arId := range arIds {
		js := s.jobManager.GetJob(arId, jobType)
		if js != nil {
			if err := s.store.SaveJobStatus(jobType, arId, *js); err != nil {
				log.Error("s.store.SaveJobStatus(jobType,arId,*js)", "err", err, "jobType", jobType, "arId", arId)
				return err
			}
		}
		// unregister job
		s.jobManager.UnregisterJob(arId, jobType)
	}

	// remove pending pool
	if err := s.store.BatchDeletePendingPool(jobType, arIds); err != nil {
		log.Error("s.store.BatchDeletePendingPool(jobType,arIds)", "err", err, "jobType", jobType, "arIds", arIds)
		return err
	}
	return nil
}

func (s *Arseeding) watcherAndCloseJobs() {
	jobs := s.jobManager.GetJobs()
	now := time.Now().Unix()
	for _, job := range jobs {
		if job.Close || job.Timestamp == 0 { // timestamp == 0  means do not start
			continue
		}
		// spend time not more than 5 minutes
		if now-job.Timestamp > 5*60 {
			if err := s.jobManager.CloseJob(job.ArId, job.JobType); err != nil {
				log.Error("watcherAndCloseJobs closeJob", "err", err, "jobId", AssembleId(job.ArId, job.JobType))
				continue
			}
		}
	}
}
