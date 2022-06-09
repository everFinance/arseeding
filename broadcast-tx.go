package arseeding

import (
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/utils"
)

func (s *Arseeding) RegisterBroadcastTx(arid string) (err error) {
	if err = s.jobManager.RegisterJob(arid, jobTypeBroadcast); err != nil {
		log.Error("Register fail", "err", err)
		return
	}

	if err = s.store.PutPendingPool(jobTypeBroadcast, arid); err != nil {
		s.jobManager.UnregisterJob(arid, jobTypeBroadcast)
		log.Error("PutPendingPool(jobTypeTxBroadcast, arTx.ID)", "err", err, "arId", arid)
		return
	}
	s.jobManager.PutToBroadcastTxChan(arid)
	return nil
}

func (s *Arseeding) RunBroadcastTx() {
	for {
		select {
		case arId := <-s.jobManager.PopBroadcastTxChan():
			go func(arId string) {
				if err := s.processBroadcastJob(arId); err != nil {
					log.Error("s.processBroadcastTxJob(arId)", "err", err, "arId", arId)
				} else {
					log.Debug("success processBroadcastTxJob", "arId", arId)
				}
				if err := s.setProcessedJobs([]string{arId}, jobTypeBroadcast); err != nil {
					log.Error("s.setProcessedJobs(arId)", "err", err, "arId", arId)
				}
			}(arId)
		}
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
