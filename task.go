package arseeding

import (
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"math/big"
)

func (s *Arseeding) runTask() {
	for {
		select {
		case jobId := <-s.taskMg.PopTkChan():
			go s.processTask(jobId)
		}
	}
}

func (s *Arseeding) processTask(taskId string) {
	arId, taskType, err := splitTaskId(taskId)
	if err != nil {
		log.Error("splitTaskId", "err", err, "taskId", taskId)
	}
	switch taskType {
	case schema.TaskTypeSync:
		err = s.syncTask(arId)
	case schema.TaskTypeBroadcast:
		err = s.broadcastTxTask(arId)
	case schema.TaskTypeBroadcastMeta:
		err = s.broadcastTxMetaTask(arId)
	}
	if err != nil {
		log.Error("process task failed", "err", err, "taskId", taskId)
		return
	}

	if err = s.setProcessedTask(arId, taskType); err != nil {
		log.Error("setProcessedTask failed", "err", err, "taskId", taskId)
	}
}

func (s *Arseeding) syncTask(arId string) (err error) {
	// 0. job manager set
	if s.taskMg.IsClosed(arId, schema.TaskTypeSync) {
		return
	}
	if err = s.taskMg.TaskBeginSet(arId, schema.TaskTypeSync, len(s.cache.GetPeers())); err != nil {
		log.Error("s.taskMg.TaskBeginSet(arId, TaskTypeSync)", "err", err, "arId", arId)
		return
	}

	err = s.fetchAndStoreTx(arId)
	if err == nil {
		s.taskMg.IncSuccessed(arId, schema.TaskTypeSync)
	}
	return err
}

func (s *Arseeding) broadcastTxTask(arId string) (err error) {
	// job manager set
	if s.taskMg.IsClosed(arId, schema.TaskTypeBroadcast) {
		log.Warn("broadcast task was closed", "arId", arId)
		return
	}
	if err = s.taskMg.TaskBeginSet(arId, schema.TaskTypeBroadcast, len(s.cache.GetPeers())); err != nil {
		log.Error("s.taskMg.TaskBeginSet(arId, TaskTypeBroadcast)", "err", err, "arId", arId)
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

	s.taskMg.BroadcastData(arId, schema.TaskTypeBroadcast, txMeta, s.cache.GetPeers(), txMetaPosted)
	return
}

func (s *Arseeding) broadcastTxMetaTask(arId string) (err error) {
	if !s.store.IsExistTxMeta(arId) {
		return ErrNotExist
	}
	txMeta, err := s.store.LoadTxMeta(arId)
	if err != nil {
		log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
		return err
	}

	if s.taskMg.IsClosed(arId, schema.TaskTypeBroadcastMeta) {
		return
	}
	if err = s.taskMg.TaskBeginSet(arId, schema.TaskTypeBroadcastMeta, len(s.cache.GetPeers())); err != nil {
		log.Error("s.taskMg.TaskBeginSet(arId, TaskTypeBroadcastMeta)", "err", err, "arId", arId)
		return
	}
	s.taskMg.BroadcastTxMeta(arId, schema.TaskTypeBroadcastMeta, txMeta, s.cache.GetPeers())
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
			arTxMeta, err = s.taskMg.GetUnconfirmedTxFromPeers(arId, schema.TaskTypeSync, s.cache.GetPeers())
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
		data, err = s.taskMg.GetTxDataFromPeers(arId, schema.TaskTypeSync, s.cache.GetPeers())
		if err != nil {
			log.Error("get data failed", "err", err, "arId", arId)
			return err
		}
	}

	// store data to local
	return setTxDataChunks(*arTxMeta, data, s.store)
}

func (s *Arseeding) setProcessedTask(arId string, tktype string) error {
	taskId := assembleTaskId(arId, tktype)

	tk := s.taskMg.GetTask(arId, tktype)
	if tk != nil {
		if err := s.store.SaveTask(taskId, *tk); err != nil {
			return err
		}
	}
	// unregister job
	s.taskMg.DelTask(arId, tktype)

	// remove pending pool
	return s.store.DelPendingPoolTaskId(taskId)
}
