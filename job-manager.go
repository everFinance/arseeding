package arseeding

import (
	"errors"
	"fmt"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"strings"
	"sync"
)

const (
	jobTypeBroadcast = "broadcast"
	jobTypeSync      = "sync"
)

type JobStatus struct {
	ArId           string `json:"arid"`
	JobType        string `json:"jobType"`
	CountSuccessed int64  `json:"countSuccessed"`
	CountFailed    int64  `json:"countFailed"`
	TotalNodes     int    `json:"totalNodes"`
	Close          bool
}

type JobManager struct {
	cap    int
	status map[string]*JobStatus // key: jobType-arId
	locker sync.RWMutex
}

func NewJobManager(cap int) *JobManager {
	return &JobManager{
		cap:    cap,
		status: make(map[string]*JobStatus),
		locker: sync.RWMutex{},
	}
}

func AssembleId(arid, jobType string) string {
	return strings.ToUpper(jobType) + "-" + arid
}

func (m *JobManager) InitJobManager(boltDb *Store, peersNum int) error {
	pendingBroadcast, err := boltDb.LoadPendingPool(jobTypeBroadcast, -1)
	if err != nil {
		return err
	}
	for _, id := range pendingBroadcast {
		if err := m.RegisterJob(id, jobTypeBroadcast, peersNum); err != nil {
			return err
		}
	}

	pendingSync, err := boltDb.LoadPendingPool(jobTypeSync, -1)
	if err != nil {
		return err
	}
	for _, id := range pendingSync {
		if err := m.RegisterJob(id, jobTypeSync, peersNum); err != nil {
			return err
		}
	}
	return nil
}

func (m *JobManager) RegisterJob(arid, jobType string, totalNodes int) (err error) {
	if m.exist(arid, jobType) {
		return errors.New("exist job")
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	if len(m.status) >= m.cap {
		err = fmt.Errorf("fully loaded")
		return
	}

	id := AssembleId(arid, jobType)
	m.status[id] = &JobStatus{
		ArId:       arid,
		JobType:    jobType,
		TotalNodes: totalNodes,
	}
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
	pNode := goar.NewShortConn()
	for _, peer := range peers {
		if j.IsClosed(arId, jobType) {
			return nil, errors.New("job closed")
		}

		pNode.SetShortConnUrl("http://" + peer)
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
	pNode := goar.NewShortConn()
	for _, peer := range peers {
		if j.IsClosed(arId, jobType) {
			return nil, errors.New("job closed")
		}
		pNode.SetShortConnUrl("http://" + peer)
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

func (j *JobManager) BroadcastData(arId, jobType string, tx *types.Transaction, peers []string, txPosted bool) error {
	pNode := goar.NewShortConn()
	for _, peer := range peers {
		pNode.SetShortConnUrl("http://" + peer)
		uploader, err := goar.CreateUploader(pNode, tx, nil)
		if err != nil {
			j.IncFailed(arId, jobType)
			continue
		}

		// Whether to broadcast txMeta
		uploader.TxPosted = txPosted
		if err = uploader.Once(); err != nil {
			log.Error("uploader.Once()", "err", err)
			j.IncFailed(arId, jobType)
			continue
		}
		// success send
		j.IncSuccessed(arId, jobType)

		// listen close status
		if j.IsClosed(arId, jobType) {
			return errors.New("job closed")
		}
	}
	return nil
}
