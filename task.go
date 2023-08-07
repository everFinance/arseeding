package arseeding

import (
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/utils"
)

func (s *Arseeding) runTask() {
	for {
		select {
		case taskId := <-s.taskMg.PopTkChan():
			go s.processTask(taskId)
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
	case schema.TaskTypeSyncManifest:
		err = s.syncManifestTask(arId)
	}

	if err != nil {
		log.Error("process task failed", "err", err, "taskId", taskId)
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

	err = s.FetchAndStoreTx(arId)
	if err == nil {
		s.taskMg.IncSuccessed(arId, schema.TaskTypeSync)
		s.taskMg.CloseTask(arId, schema.TaskTypeSync)
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
		if err = s.FetchAndStoreTx(arId); err != nil {
			log.Error("processBroadcast FetchAndStoreTx failed", "err", err, "arId", arId)
			return err
		}
	}

	txMeta, err := s.store.LoadTxMeta(arId)
	if err != nil {
		log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
		return err
	}
	txData, err := getArTxData(txMeta.DataRoot, txMeta.DataSize, s.store)
	if err != nil {
		if err == schema.ErrNotExist {
			if err = s.FetchAndStoreTx(arId); err != nil {
				log.Error("processBroadcast FetchAndStoreTx failed", "err", err, "arId", arId)
				return err
			}
		} else {
			log.Error("getDataByGW(txMeta.DataRoot,txMeta.DataSize,s.store)", "err", err, "arId", arId)
			return err
		}
		txData, err = getArTxData(txMeta.DataRoot, txMeta.DataSize, s.store)
		if err != nil {
			log.Error("get data failed", "err", err, "arId", arId)
			return err
		}
	}

	txMetaPosted := true
	// check this tx whether on chain
	_, err = s.arCli.GetTransactionStatus(arId)
	if err == goar.ErrPendingTx || err == goar.ErrNotFound {
		txMetaPosted = false
		err = nil
	}
	// generate tx chunks
	utils.PrepareChunks(txMeta, txData, len(txData))
	txMeta.Data = utils.Base64Encode(txData)

	s.taskMg.BroadcastData(arId, schema.TaskTypeBroadcast, txMeta, s.cache.GetPeers(), txMetaPosted)
	return
}

func (s *Arseeding) broadcastTxMetaTask(arId string) (err error) {
	if !s.store.IsExistTxMeta(arId) {
		return schema.ErrNotExist
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

func (s *Arseeding) syncManifestTask(arId string) (err error) {

	if s.taskMg.IsClosed(arId, schema.TaskTypeSyncManifest) {
		return
	}

	err = syncManifestData(arId, s)

	if err == nil {
		s.taskMg.IncSuccessed(arId, schema.TaskTypeSyncManifest)
		closeErr := s.taskMg.CloseTask(arId, schema.TaskTypeSyncManifest)

		if closeErr != nil {
			log.Error("s.taskMg.CloseTask(arId, schema.TaskTypeSyncManifest)", "err", closeErr, "arId", arId)
		}
	}
	return err

}
