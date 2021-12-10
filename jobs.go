package seeding

import (
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"math/big"
)

func (s *Server) runJobs() {
	for {
		select {
		case arId := <-s.jobManager.JobCh:
			if s.jobManager.IsClosed(arId) {
				continue
			}

			jobType := s.jobManager.GetJob(arId).JobType
			switch jobType {
			case jobTypeBroadcast:
				go func() {
					if err := s.processBroadcastJob(arId); err != nil {
						log.Error("processBroadcastJob", "err", err, "arId", arId)
					}
				}()

			case jobTypeSync:
				go func() {
					if err := s.processSyncJob(arId); err != nil {
						log.Error("processSyncJob", "err", err, "arId", arId)
						s.jobManager.IncFailed(arId)
					} else {
						s.jobManager.IncSuccessed(arId)
					}
				}()

			default:
				log.Error("not support job type", "type", jobType)
			}
		}
	}
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

	// generate tx chunks
	utils.PrepareChunks(txMeta, txData)
	txMeta.Data = utils.Base64Encode(txData)

	pNode := goar.NewShortConn()
	for _, peer := range s.peers {
		pNode.SetShortConnUrl("http://" + peer)
		uploader, err := goar.CreateUploader(pNode, txMeta, nil)
		if err != nil {
			s.jobManager.IncFailed(arId)
			continue
		}

		if err = uploader.Once(); err != nil {
			s.jobManager.IncFailed(arId)
			continue
		}
		// success send
		s.jobManager.IncSuccessed(arId)

		// listen close status
		if s.jobManager.IsClosed(arId) {
			return nil
		}
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
			arTxMeta, err = s.arCli.GetUnconfirmedTxFromPeers(arId)
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
		return nil
	}

	// get data
	var data []byte
	data, err = getData(arTxMeta.DataRoot, arTxMeta.DataSize, s.store) // get data from local
	if err == nil {
		return nil // lcoal exist data
	} else {
		// need get tx data from arweave network
		data, err = s.arCli.GetTransactionDataByGateway(arId)
		if err != nil {
			data, err = s.arCli.GetTxDataFromPeers(arId)
			if err != nil {
				log.Error("get data failed", "err", err, "arId", arId)
				return err
			}
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
