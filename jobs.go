package arseeding

import (
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/panjf2000/ants/v2"
	"math/big"
	"sync"
)

func (s *Server) runJobs() {
	s.scheduler.Every(1).Minute().SingletonMode().Do(s.updatePeers)
	s.scheduler.Every(2).Seconds().SingletonMode().Do(s.runBroadcastJobs)
	s.scheduler.Every(2).Seconds().SingletonMode().Do(s.runSyncJobs)

	s.scheduler.StartAsync()
}

func (s *Server) updatePeers() {
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

func (s *Server) runBroadcastJobs() {
	arIds, err := s.store.LoadPendingPool(jobTypeBroadcast, 20)
	if err != nil {
		log.Error("s.store.LoadPendingPool(jobTypeBroadcast, 20)", "err", err)
		return
	}
	if len(arIds) == 0 {
		return
	}

	log.Debug("load jobTypeBroadcast pending pool", "number", len(arIds))
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		defer wg.Done()
		arId := i.(string)
		if s.jobManager.IsClosed(arId, jobTypeBroadcast) {
			return
		}
		job := s.jobManager.GetJob(arId, jobTypeBroadcast)
		if job == nil {
			return
		}

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

	// save jobStatus to db and unregister job
	for _, arId := range arIds {
		js := s.jobManager.GetJob(arId, jobTypeBroadcast)
		if js == nil {
			panic(err)
		}
		if err := s.store.SaveJobStatus(jobTypeBroadcast, arId, *js); err != nil {
			log.Error("s.store.SaveJobStatus(jobTypeBroadcast,arId,*js)", "err", err, "arId", arId)
			panic(err)
		} else {
			// unregister job
			s.jobManager.UnregisterJob(arId, jobTypeBroadcast)
		}
	}

	// remove pending pool
	if err := s.store.BatchDeletePendingPool(jobTypeBroadcast, arIds); err != nil {
		log.Error("s.store.BatchDeletePendingPool(jobTypeBroadcast,arIds)", "err", err, "arIds", arIds)
		log.Debug("run broadcast jobs failed", "broadcastJobs number", len(arIds))
		return
	}
	log.Debug("run broadcast jobs success", "broadcastJobs number", len(arIds))
}

func (s *Server) runSyncJobs() {
	arIds, err := s.store.LoadPendingPool(jobTypeSync, 50)
	if err != nil {
		log.Error("s.store.LoadPendingPool(jobTypeSync, 50)", "err", err)
		return
	}

	if len(arIds) == 0 {
		return
	}

	log.Debug("load jobTypeSync pending pool", "number", len(arIds))
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(20, func(i interface{}) {
		defer wg.Done()
		arId := i.(string)
		if s.jobManager.IsClosed(arId, jobTypeSync) {
			return
		}
		job := s.jobManager.GetJob(arId, jobTypeSync)
		if job == nil {
			return
		}

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

	// save jobStatus to db and unregister job
	for _, arId := range arIds {
		js := s.jobManager.GetJob(arId, jobTypeSync)
		if js == nil {
			panic(err)
		}
		if err := s.store.SaveJobStatus(jobTypeSync, arId, *js); err != nil {
			log.Error("s.store.SaveJobStatus(jobTypeSync,arId,*js)", "err", err, "arId", arId)
			panic(err)
		} else {
			// unregister job
			s.jobManager.UnregisterJob(arId, jobTypeSync)
		}
	}

	// remove pending pool
	if err := s.store.BatchDeletePendingPool(jobTypeSync, arIds); err != nil {
		log.Error("s.store.BatchDeletePendingPool(jobTypeSync, arIds)", "err", err)
		log.Debug("run sync jobs failed", "syncJobs number", len(arIds))
		return
	}
	log.Debug("run sync jobs success", "syncJobs number", len(arIds))
}

func (s *Server) processBroadcastJob(arId string) (err error) {
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
	}
	// generate tx chunks
	utils.PrepareChunks(txMeta, txData)
	txMeta.Data = utils.Base64Encode(txData)

	if err := s.jobManager.BroadcastData(arId, jobTypeBroadcast, txMeta, s.peers, txMetaPosted); err != nil {
		log.Error("s.jobManager.BroadcastData(arId,txMeta,s.peers)", "err", err)
		return err
	}
	return
}

func (s *Server) processSyncJob(arId string) (err error) {
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
	if len(data) == 0 {
		return fmt.Errorf("data can not be null; arId: %s", arId)
	}
	chunks, err := generateChunks(*arTxMeta, data)
	if err != nil {
		return err
	}
	// save chunks
	for _, chunk := range chunks {
		if err := storeChunk(*chunk, s.store); err != nil {
			log.Error("storeChunk(*chunk,s.store)", "err", err, "arId", arId, "chunk", *chunk)
			return err
		}
	}
	return nil
}
