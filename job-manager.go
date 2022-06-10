package arseeding

import (
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"strings"
	"sync"
	"time"
)

const (
	jobTypeBroadcast     = "broadcast"      // include tx and tx data
	jobTypeMetaBroadcast = "broadcast_meta" //  not include tx data
	jobTypeSync          = "sync"
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
	status map[string]*JobStatus // key: jobType-arId
	txChan chan string           // val: jobType-arId
	locker sync.RWMutex
}

func NewJM() *JobManager {
	return &JobManager{
		status: make(map[string]*JobStatus),
		txChan: make(chan string),
		locker: sync.RWMutex{},
	}
}

func (m *JobManager) InitJM(boltDb *Store) error {
	pendingBroadcastMeta, err := boltDb.LoadPendingPool(jobTypeMetaBroadcast, -1)
	if err != nil {
		return err
	}
	log.Debug("broadcastSubmit num", "pending num", len(pendingBroadcastMeta))
	for _, arId := range pendingBroadcastMeta {
		m.AddJob(arId, jobTypeMetaBroadcast)
		go m.PutToChan(arId, jobTypeMetaBroadcast)
	}

	pendingBroadcast, err := boltDb.LoadPendingPool(jobTypeBroadcast, -1)
	if err != nil {
		return err
	}
	log.Debug("broadcast num", "pending num", len(pendingBroadcast), "arIds", pendingBroadcast)
	for _, arId := range pendingBroadcast {
		m.AddJob(arId, jobTypeBroadcast)
		go m.PutToChan(arId, jobTypeBroadcast)
	}

	pendingSync, err := boltDb.LoadPendingPool(jobTypeSync, -1)
	if err != nil {
		return err
	}
	for _, arId := range pendingSync {
		m.AddJob(arId, jobTypeSync)
		go m.PutToChan(arId, jobTypeSync)
	}

	return nil
}

func (m *JobManager) AddJob(arid, jobType string) {
	if m.exist(arid, jobType) {
		return
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	m.status[assembleId(arid, jobType)] = &JobStatus{
		ArId:    arid,
		JobType: jobType,
	}
}

func (m *JobManager) DelJob(arid, jobType string) {
	id := assembleId(arid, jobType)
	delete(m.status, id)
}

func (m *JobManager) exist(arid, jobType string) bool {
	id := assembleId(arid, jobType)
	_, ok := m.status[id]
	return ok
}

func (m *JobManager) IncSuccessed(arid, jobType string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	id := assembleId(arid, jobType)
	if s, ok := m.status[id]; ok {
		s.CountSuccessed += 1
	}
}

func (m *JobManager) IncFailed(arid, jobType string) {
	m.locker.Lock()
	defer m.locker.Unlock()
	id := assembleId(arid, jobType)
	if s, ok := m.status[id]; ok {
		s.CountFailed += 1
	}
}

func (m *JobManager) JobBeginSet(arid, jobType string, totalNodes int) error {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := assembleId(arid, jobType)
	job, ok := m.status[id]
	if !ok {
		return ErrNotFound
	}
	job.Timestamp = time.Now().Unix()
	job.TotalNodes = totalNodes
	return nil
}

func (m *JobManager) GetJob(arid, jobType string) *JobStatus {
	m.locker.RLock()
	defer m.locker.RUnlock()
	id := assembleId(arid, jobType)
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
	id := assembleId(arid, jobType)
	job, ok := m.status[id]
	if ok {
		job.Close = true
	} else {
		return ErrNotFound
	}
	return nil
}

func (m *JobManager) IsClosed(arid, jobType string) bool {
	id := assembleId(arid, jobType)
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
			return nil, ErrJobClosed
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
			return nil, ErrJobClosed
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

	return nil, ErrFetchData
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

func (j *JobManager) PopChan() <-chan string {
	return j.txChan
}

func (j *JobManager) PutToChan(arId, jobType string) {
	j.txChan <- assembleId(arId, jobType)
}

func assembleId(arid, jobType string) string {
	return strings.ToLower(jobType) + "-" + arid
}
