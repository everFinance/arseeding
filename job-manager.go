package arseeding

import (
	"errors"
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"strings"
	"sync"
	"time"
)

const (
	jobTypeBroadcast         = "broadcast"           // include tx and tx data
	jobTypeSubmitTxBroadcast = "submit-tx-broadcast" //  not include tx data
	jobTypeSync              = "sync"
)

type JobStatus struct {
	ArId           string `json:"arId"`
	JobType        string `json:"jobType"`
	CountSuccessed int64  `json:"countSuccessed"`
	CountFailed    int64  `json:"countFailed"`
	TotalNodes     int    `json:"totalNodes"`
	Timestamp      int64  `json:"timestamp"` // begin timestamp
	Close          bool   `json:"close"`
}

type JobManager struct {
	cap                   int
	status                map[string]*JobStatus // key: jobType-arId
	broadcastSubmitTxChan chan string
	broadcastTxChan       chan string
	syncTxChan            chan string
	locker                sync.RWMutex
}

func NewJM(cap int) *JobManager {
	return &JobManager{
		cap:                   cap,
		status:                make(map[string]*JobStatus),
		locker:                sync.RWMutex{},
	}
}

func AssembleId(arid, jobType string) string {
	return strings.ToUpper(jobType) + "-" + arid
}

func (m *JobManager) InitJM(boltDb *Store) error {
	pendingBroadcastSubmitTx, err := boltDb.LoadPendingPool(jobTypeSubmitTxBroadcast, -1)
	if err != nil {
		return err
	}
	m.broadcastSubmitTxChan = make(chan string, len(pendingBroadcastSubmitTx))
	for _, arId := range pendingBroadcastSubmitTx {
		m.PutToBroadcastSubmitTxChan(arId)
		m.AddJob(arId, jobTypeSubmitTxBroadcast)
	}

	pendingBroadcast, err := boltDb.LoadPendingPool(jobTypeBroadcast,-1)
	if err != nil {
		return err
	}
	m.broadcastTxChan = make(chan string, len(pendingBroadcast))
	for _, arId := range pendingBroadcastSubmitTx {
		m.PutToBroadcastTxChan(arId)
		m.AddJob(arId, jobTypeBroadcast)
	}

	pendingSync, err := boltDb.LoadPendingPool(jobTypeSync,-1)
	if err != nil {
		return err
	}
	m.syncTxChan =  make(chan string, len(pendingSync))
	for _, arId := range pendingSync {
		m.PutToSyncTxChan(arId)
		m.AddJob(arId, jobTypeSync)
	}

	return nil
}

func (m *JobManager) RegisterJob(arid, jobType string) (err error) {
	if m.exist(arid, jobType) {
		return errors.New("exist job")
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	if len(m.status) >= m.cap {
		err = fmt.Errorf("fully loaded")
		return
	}
	m.AddJob(arid, jobType)
	return
}

func (m *JobManager) exist(arid, jobType string) bool {
	id := AssembleId(arid, jobType)
	_, ok := m.status[id]
	return ok
}

func (m *JobManager) IncSuccessed(arid, jobType string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	id := AssembleId(arid, jobType)
	if s, ok := m.status[id]; ok {
		s.CountSuccessed += 1
	}
}

func (m *JobManager) IncFailed(arid, jobType string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	id := AssembleId(arid, jobType)
	if s, ok := m.status[id]; ok {
		s.CountFailed += 1
	}
}

func (m *JobManager) UnregisterJob(arid, jobType string) {
	id := AssembleId(arid, jobType)
	delete(m.status, id)
}

func (m *JobManager) AddJob(arid, jobType string) {
	id := AssembleId(arid, jobType)
	m.status[id] = &JobStatus{
		ArId:    arid,
		JobType: jobType,
	}
	return
}

func (m *JobManager) JobBeginSet(arid, jobType string, totalNodes int) error {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := AssembleId(arid, jobType)
	job, ok := m.status[id]
	if !ok {
		return errors.New("not found")
	}
	job.Timestamp = time.Now().Unix()
	job.TotalNodes = totalNodes
	return nil
}

func (m *JobManager) GetJob(arid, jobType string) *JobStatus {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := AssembleId(arid, jobType)
	job := JobStatus{}
	j, ok := m.status[id]
	if ok {
		job = *j
		return &job
	}
	return nil
}

func (m *JobManager) CloseJob(arid, jobType string) error {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := AssembleId(arid, jobType)
	job, ok := m.status[id]
	if ok {
		job.Close = true
	} else {
		return errors.New("not found")
	}
	return nil
}

func (m *JobManager) IsClosed(arid, jobType string) bool {
	id := AssembleId(arid, jobType)
	job, ok := m.status[id]
	if ok {
		return job.Close
	}
	return false
}

func (m *JobManager) GetJobs() (jobs map[string]JobStatus) {
	m.locker.RLock()
	defer m.locker.RUnlock()

	jobs = make(map[string]JobStatus, len(m.status))
	for id, job := range m.status {
		jobs[id] = *job
	}
	return
}

func (j *JobManager) GetUnconfirmedTxFromPeers(arId, jobType string, peers []string) (*types.Transaction, error) {
	pNode := goar.NewTempConn()
	for _, peer := range peers {
		if j.IsClosed(arId, jobType) {
			return nil, errors.New("job closed")
		}

		pNode.SetTempConnUrl("http://" + peer)
		tx, err := pNode.GetUnconfirmedTx(arId)
		if err != nil {
			fmt.Printf("get tx error:%v, peer: %s, arTx: %s\n", err, peer, arId)
			continue
		}
		fmt.Printf("success get unconfirmed tx; peer: %s, arTx: %s\n", peer, arId)
		return tx, nil
	}

	return nil, fmt.Errorf("get unconfirmed tx failed; arId: %s", arId)
}

func (j *JobManager) GetTxDataFromPeers(arId, jobType string, peers []string) ([]byte, error) {
	pNode := goar.NewTempConn()
	for _, peer := range peers {
		if j.IsClosed(arId, jobType) {
			return nil, errors.New("job closed")
		}
		pNode.SetTempConnUrl("http://" + peer)
		data, err := pNode.DownloadChunkData(arId)
		if err != nil {
			log.Error("get tx data", "err", err, "peer", peer)
			j.IncFailed(arId, jobType)
			continue
		}
		j.IncSuccessed(arId, jobType)
		return data, nil
	}

	return nil, errors.New("get tx data from peers failed")
}

func (j *JobManager) BroadcastTxMeta(arId, jobType string, tx *types.Transaction, peers []string) {
	pNode := goar.NewTempConn()
	for _, peer := range peers {
		pNode.SetTempConnUrl("http://" + peer)
		status, code, _ := pNode.SubmitTransaction(&types.Transaction{
			Format:    tx.Format,
			ID:        tx.ID,
			LastTx:    tx.LastTx,
			Owner:     tx.Owner,
			Tags:      tx.Tags,
			Target:    tx.Target,
			Quantity:  tx.Quantity,
			Data:      "",
			DataSize:  tx.DataSize,
			DataRoot:  tx.DataRoot,
			Reward:    tx.Reward,
			Signature: tx.Signature,
		})
		if code != 200 && code != 208 {
			log.Debug("BroadcastTxMeta submit tx failed", "err", status, "arId", arId, "code", code, "peerIp", peer)
			j.IncFailed(arId, jobType)
		} else {
			// success send
			j.IncSuccessed(arId, jobType)
		}

		// listen close status
		if j.IsClosed(arId, jobType) {
			log.Debug("job is closed", "arId", arId, "jobType", jobType)
			return
		}
	}
	// close job
	j.CloseJob(arId, jobType)
}

func (j *JobManager) BroadcastData(arId, jobType string, tx *types.Transaction, peers []string, txPosted bool) {
	pNode := goar.NewTempConn()
	for _, peer := range peers {
		pNode.SetTempConnUrl("http://" + peer)
		uploader, err := goar.CreateUploader(pNode, tx, nil)
		if err != nil {
			j.IncFailed(arId, jobType)
			continue
		}

		// post tx
		if !txPosted {
			status, code, _ := pNode.SubmitTransaction(&types.Transaction{
				Format:    tx.Format,
				ID:        tx.ID,
				LastTx:    tx.LastTx,
				Owner:     tx.Owner,
				Tags:      tx.Tags,
				Target:    tx.Target,
				Quantity:  tx.Quantity,
				Data:      "",
				DataSize:  tx.DataSize,
				DataRoot:  tx.DataRoot,
				Reward:    tx.Reward,
				Signature: tx.Signature,
			})
			if code != 200 && code != 208 {
				log.Debug("BroadcastData submit tx failed", "err", status, "arId", arId, "code", code, "peerIp", peer)
			}
		}

		uploader.TxPosted = true // only broadcast tx data
		if err = uploader.Once(); err != nil {
			log.Debug("uploader.Once()", "err", err, "peerIp", peer)
			j.IncFailed(arId, jobType)
		} else {
			// success send
			j.IncSuccessed(arId, jobType)
		}

		// listen close status
		if j.IsClosed(arId, jobType) {
			log.Debug("job is closed", "arId", arId, "jobType", jobType)
			return
		}
	}
	j.CloseJob(arId, jobType)
	return
}

func (j *JobManager) PopBroadcastSubmitTxChan() <-chan string {
	return j.broadcastSubmitTxChan
}

func (j *JobManager) PutToBroadcastSubmitTxChan(txId string) {
	j.broadcastSubmitTxChan <- txId
}

func (j *JobManager) PopBroadcastTxChan() <-chan string {
	return j.broadcastTxChan
}

func (j *JobManager) PutToBroadcastTxChan(arid string) {
	j.broadcastTxChan <- arid
}

func (j *JobManager) PopSyncTxChan() <-chan string {
	return j.syncTxChan
}

func (j *JobManager) PutToSyncTxChan(arid string) {
	j.syncTxChan <- arid
}
