package arseeding

import (
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
)

func (s *Arseeding) RunBroadcastTxMeta() {
	for {
		select {
		case arId := <-s.jobManager.PopBroadcastTxMetaChan():
			go func(arId string) {
				if err := s.processBroadcastTxMetaJob(arId); err != nil {
					log.Error("s.processBroadcastTxMetaJob(arId)", "err", err, "arId", arId)
				} else {
					log.Debug("success processBroadcastTxMetaJob", "arId", arId)
				}
				if err := s.setProcessedJobs([]string{arId}, jobTypeTxMetaBroadcast); err != nil {
					log.Error("s.setProcessedJobs(arId)", "err", err, "arId", arId)
				}
			}(arId)

		}
	}
}

func (s *Arseeding) processBroadcastTxMetaJob(arId string) (err error) {
	if !s.store.IsExistTxMeta(arId) {
		return ErrNotExist
	}
	txMeta, err := s.store.LoadTxMeta(arId)
	if err != nil {
		log.Error("s.store.LoadTxMeta(arId)", "err", err, "arId", arId)
		return err
	}

	if s.jobManager.IsClosed(arId, jobTypeTxMetaBroadcast) {
		return
	}
	if err = s.jobManager.JobBeginSet(arId, jobTypeTxMetaBroadcast, len(s.cache.GetPeers())); err != nil {
		log.Error("s.jobManager.JobBeginSet(arId, jobTypeTxMetaBroadcast)", "err", err, "arId", arId)
		return
	}
	s.jobManager.BroadcastTxMeta(arId, jobTypeTxMetaBroadcast, txMeta, s.cache.GetPeers())
	return
}

func (s *Arseeding) broadcastSubmitTx(arTx types.Transaction) error {
	if arTx.ID == "" {
		return ErrNullArId
	}
	// save tx to local
	if err := s.processSubmitTx(arTx); err != nil {
		log.Error("s.processSubmitTx", "err", err)
		return err
	}

	// add broadcast submit arTx
	s.jobManager.AddJob(arTx.ID, jobTypeTxMetaBroadcast)

	if err := s.store.PutPendingPool(jobTypeTxMetaBroadcast, arTx.ID); err != nil {
		s.jobManager.UnregisterJob(arTx.ID, jobTypeTxMetaBroadcast)
		log.Error("PutPendingPool(jobTypeTxMetaBroadcast, arTx.ID)", "err", err, "arId", arTx.ID)
		return err
	}

	// put channel
	s.jobManager.PutToBroadcastTxMetaChan(arTx.ID)

	return nil
}

func (s *Arseeding) processSubmitTx(arTx types.Transaction) error {
	// 1. verify ar tx
	if err := utils.VerifyTransaction(arTx); err != nil {
		log.Error("utils.VerifyTransaction(arTx)", "err", err, "arTx", arTx.ID)
		return err
	}

	// 2. check meta exist
	if s.store.IsExistTxMeta(arTx.ID) {
		return ErrExistTx
	}

	// 3. save tx meta
	if err := s.store.SaveTxMeta(arTx); err != nil {
		log.Error("s.store.SaveTxMeta(arTx)", "err", err, "arTx", arTx.ID)
		return err
	}

	s.submitLocker.Lock()
	defer s.submitLocker.Unlock()

	// 4. check whether update allDataEndOffset
	if s.store.IsExistTxDataEndOffset(arTx.DataRoot, arTx.DataSize) {
		return nil
	}
	// add txDataEndOffset
	if err := s.syncAddTxDataEndOffset(arTx.DataRoot, arTx.DataSize); err != nil {
		log.Error("syncAddTxDataEndOffset(s.store,arTx.DataRoot,arTx.DataSize)", "err", err, "arTx", arTx.ID)
		return err
	}

	// 5. save tx data chunk if exist
	if len(arTx.Data) > 0 {
		// set chunks
		dataBy, err := utils.Base64Decode(arTx.Data)
		if err != nil {
			log.Error("utils.Base64Decode(arTx.Data)", "err", err, "data", arTx.Data)
			return err
		}
		if err := setTxDataChunks(arTx, dataBy, s.store); err != nil {
			return err
		}
	}
	log.Debug("success process a new arTx", "arTx", arTx.ID)
	return nil
}
