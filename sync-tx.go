package arseeding

import (
	"fmt"
	"github.com/everFinance/goar/types"
	"math/big"
)

func (s *Arseeding) RegisterSyncTx(arid string) (err error) {
	if err = s.jobManager.RegisterJob(arid, jobTypeSync); err != nil {
		log.Error("Register fail", "err", err)
		return
	}

	if err = s.store.PutPendingPool(jobTypeSync, arid); err != nil {
		s.jobManager.UnregisterJob(arid, jobTypeSync)
		log.Error("PutPendingPool(jobTypeSync, arTx.ID)", "err", err, "arId", arid)
		return
	}
	s.jobManager.PutToSyncTxChan(arid)
	return nil
}

func (s *Arseeding) RunSyncTx() {
	for {
		select {
		case arId := <-s.jobManager.PopSyncTxChan():
			go func(arId string) {
				if err := s.processSyncJob(arId); err != nil {
					log.Error("s.processSyncTxJob(arId)", "err", err, "arId", arId)
				} else {
					log.Debug("success processSyncTxJob", "arId", arId)
				}
				if err := s.setProcessedJobs([]string{arId}, jobTypeSync); err != nil {
					log.Error("s.setProcessedJobs(arId)", "err", err, "arId", arId)
				}
			}(arId)
		}
	}
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

	err = s.FetchAndStoreTx(arId)
	if err == nil {
		s.jobManager.IncSuccessed(arId, jobTypeSync)
	}
	return err
}

func (s *Arseeding) FetchAndStoreTx(arId string) (err error) {
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
		data, err = s.jobManager.GetTxDataFromPeers(arId, jobTypeSync, s.peers)
		if err != nil {
			log.Error("get data failed", "err", err, "arId", arId)
			return err
		}
	}

	// store data to local
	return setTxDataChunks(*arTxMeta, data, s.store)
}
