package arseeding

import (
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"math/big"
	"time"
)

func (s *Server) runJobs() {
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.updateAnchor)
	s.scheduler.Every(2).Minute().SingletonMode().Do(s.updatePrice)
	s.scheduler.Every(30).Seconds().SingletonMode().Do(s.updateInfo)
	s.scheduler.Every(30).Minute().SingletonMode().Do(s.updatePeerList)

	s.scheduler.Every(1).Minute().SingletonMode().Do(s.updatePeers)
	s.scheduler.Every(5).Seconds().SingletonMode().Do(s.watcherAndCloseJobs)

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

func (s *Server) processBroadcastJob(arId string) (err error) {
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
		if err = s.fetchAndStoreTx(arId); err != nil {
			log.Error("processBroadcast fetchAndStoreTx failed", "err", err, "arId", arId)
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

func (s *Server) processSyncJob(arId string) (err error) {
	// 0. job manager set
	if s.jobManager.IsClosed(arId, jobTypeSync) {
		return
	}
	if err = s.jobManager.JobBeginSet(arId, jobTypeSync, len(s.peers)); err != nil {
		log.Error("s.jobManager.JobBeginSet(arId, jobTypeSync)", "err", err, "arId", arId)
		return
	}

	err = s.fetchAndStoreTx(arId)
	if err == nil {
		s.jobManager.IncSuccessed(arId, jobTypeSync)
	}
	return err
}

func (s *Server) fetchAndStoreTx(arId string) (err error) {
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

func (s *Server) setProcessedJobs(arIds []string, jobType string) error {
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

func (s *Server) watcherAndCloseJobs() {
	jobs := s.jobManager.GetJobs()
	now := time.Now().Unix()
	for _, job := range jobs {
		if job.Close || job.Timestamp == 0 { // timestamp == 0  means do not start
			continue
		}
		// spend time not more than 30 minutes
		if now-job.Timestamp > 30*60 {
			if err := s.jobManager.CloseJob(job.ArId, job.JobType); err != nil {
				log.Error("watcherAndCloseJobs closeJob", "err", err, "jobId", AssembleId(job.ArId, job.JobType))
				continue
			}
		}
	}
}

func (s *Server) updateAnchor() {
	anchor, err := s.arCli.GetTransactionAnchor()
	if err != nil {
		pNode := goar.NewTempConn()
		for _, peer := range s.peers {
			pNode.SetTempConnUrl("http://" + peer)
			anchor, err = pNode.GetTransactionAnchor()
			if err == nil && len(anchor) > 0 {
				break
			}
		}
	}
	s.cache.UpdateAnchor(anchor)
}

// update arweave network info

func (s *Server) updateInfo() {
	info, err := s.arCli.GetInfo()
	if err != nil {
		pNode := goar.NewTempConn()
		for _, peer := range s.peers {
			pNode.SetTempConnUrl("http://" + peer)
			info, err = pNode.GetInfo()
			if err == nil && info != nil {
				break
			}
		}
	}
	if err != nil {
		return
	}
	s.cache.UpdateInfo(info)
}

func (s *Server) updatePrice() {
	// base price /price/0  datasize = 0,data = nil
	var basePrice, deltaPrice int64
	var err1, err2 error

	littleData := make([]byte, 1)
	basePrice, err1 = s.arCli.GetTransactionPrice(nil, nil)
	deltaPrice, err2 = s.arCli.GetTransactionPrice(littleData, nil)
	if err1 != nil || err2 != nil {
		pNode := goar.NewTempConn()
		for _, peer := range s.peers {
			pNode.SetTempConnUrl("http://" + peer)
			basePrice, err1 = pNode.GetTransactionPrice(nil, nil)
			deltaPrice, err2 = pNode.GetTransactionPrice(littleData, nil)
			if err1 == nil && err2 == nil { // fetch price from one peer
				break
			}
		}
	}

	if err1 != nil || err2 != nil {
		return
	}
	s.cache.UpdatePrice(TxPrice{basePrice, deltaPrice - basePrice})
}

// update peer list, check peer available, store in db
// TODO update peerList concurrency
func (s *Server) updatePeerList() {
	visPeer := make(map[string]bool, 0) // record already handled peer
	updatedPeers := make([]string, 0)
	pNode := goar.NewTempConn()
	for _, peer := range s.peers {
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
	s.peers = updatedPeers
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
