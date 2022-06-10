package arseeding

import (
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"math/big"
	"strings"
)

func (s *Arseeding) runJM() {
	for {
		select {
		case jobId := <-s.jobManager.PopChan():
			go s.processJob(jobId)
		}
	}
}

func (s *Arseeding) processJob(jobId string) {
	ss := strings.SplitN(jobId, "-", 2)
	if len(ss) != 2 {
		log.Error("jobId incorrect", "jobId", jobId)
		return
	}
	var err error
	jobTp, arId := ss[0], ss[1]
	switch jobTp {
	case jobTypeSync:
		err = s.syncJob(arId)
	case jobTypeBroadcast:
		err = s.broadcastTxJob(arId)
	case jobTypeMetaBroadcast:
		err = s.broadcastTxMetaJob(arId)
	}
	if err != nil {
		log.Error("process job failed", "err", err, "jobId", jobId)
		return
	}

	if err = s.setProcessedJobs([]string{arId}, jobTp); err != nil {
		log.Error("setProcessedJobs failed", "err", err, "jobId", jobId)
	}
}

func (s *Arseeding) syncJob(arId string) (err error) {
	// 0. job manager set
	if s.jobManager.IsClosed(arId, jobTypeSync) {
		return
	}
	if err = s.jobManager.JobBeginSet(arId, jobTypeSync, len(s.cache.GetPeers())); err != nil {
		log.Error("s.jobManager.JobBeginSet(arId, jobTypeSync)", "err", err, "arId", arId)
		return
	}

	err = s.fetchAndStoreTx(arId)
	if err == nil {
		s.jobManager.IncSuccessed(arId, jobTypeSync)
	}
	return err
}

func (s *Arseeding) broadcastTxJob(arId string) (err error) {
	// job manager set
	if s.jobManager.IsClosed(arId, jobTypeBroadcast) {
		log.Warn("broadcast job was closed", "arId", arId)
		return
	}
	if err = s.jobManager.JobBeginSet(arId, jobTypeBroadcast, len(s.cache.GetPeers())); err != nil {
		log.Error("s.jobManager.JobBeginSet(arId, jobTypeBroadcast)", "err", err, "arId", arId)
		return
	}

	if !s.store.IsExistTxMeta(arId) { // if the arId not exist local db, then wait sync to local
		if err = s.fetchAndStoreTx(arId); err != nil {
			log.Error("processBroadcast FetchAndStoreTx failed", "err", err, "arId", arId)
			return err
		}
	}

	txMeta, err := s.store.LoadTxMeta(arId)
	if err != nil {
		log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
		return err
	}
	txData, err := getData(txMeta.DataRoot, txMeta.DataSize, s.store)
	if err != nil {
		log.Error("getDataByGW(txMeta.DataRoot,txMeta.DataSize,s.store)", "err", err, "arId", arId)
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

	s.jobManager.BroadcastData(arId, jobTypeBroadcast, txMeta, s.cache.GetPeers(), txMetaPosted)
	return
}

func (s *Arseeding) broadcastTxMetaJob(arId string) (err error) {
	if !s.store.IsExistTxMeta(arId) {
		return ErrNotExist
	}
	txMeta, err := s.store.LoadTxMeta(arId)
	if err != nil {
		log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
		return err
	}

	if s.jobManager.IsClosed(arId, jobTypeMetaBroadcast) {
		return
	}
	if err = s.jobManager.JobBeginSet(arId, jobTypeMetaBroadcast, len(s.cache.GetPeers())); err != nil {
		log.Error("s.jobManager.JobBeginSet(arId, jobTypeMetaBroadcast)", "err", err, "arId", arId)
		return
	}
	s.jobManager.BroadcastTxMeta(arId, jobTypeMetaBroadcast, txMeta, s.cache.GetPeers())
	return
}

func (s *Arseeding) fetchAndStoreTx(arId string) (err error) {
	// 1. sync arTxMeta
	arTxMeta := &types.Transaction{}
	arTxMeta, err = s.store.LoadTxMeta(arId)
	if err != nil {
		// get txMeta from arweave network
		arTxMeta, err = s.arCli.GetUnconfirmedTx(arId) // this api can return all tx (unconfirmed and confirmed)
		if err != nil {
			// get tx from peers
			arTxMeta, err = s.jobManager.GetUnconfirmedTxFromPeers(arId, jobTypeSync, s.cache.GetPeers())
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
		// not need to sync tx data
		return nil
	}

	// get data
	var data []byte
	data, err = getData(arTxMeta.DataRoot, arTxMeta.DataSize, s.store) // get data from local
	if err == nil {
		return nil // local exist data
	}
	// need get tx data from arweave network
	data, err = s.arCli.GetTransactionDataByGateway(arId)
	if err != nil {
		data, err = s.jobManager.GetTxDataFromPeers(arId, jobTypeSync, s.cache.GetPeers())
		if err != nil {
			log.Error("get data failed", "err", err, "arId", arId)
			return err
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
		s.jobManager.DelJob(arId, jobType)
	}

	// remove pending pool
	if err := s.store.BatchDeletePendingPool(jobType, arIds); err != nil {
		log.Error("s.store.BatchDeletePendingPool(jobType,arIds)", "err", err, "jobType", jobType, "arIds", arIds)
		return err
	}
	return nil
}
