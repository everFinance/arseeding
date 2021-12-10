package seeding

import (
	"errors"
	"fmt"
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
	TotalNodes     int64  `json:"totalNodes"`
	NumOfProcessed int64  `json:"numProcessed"` // todo ignore
	Close          bool
}

type JobManager struct {
	cap    int
	status map[string]*JobStatus
	JobCh  chan string
	locker sync.RWMutex
}

func NewJobManager(cap int) *JobManager {
	return &JobManager{
		cap:    cap,
		status: make(map[string]*JobStatus),
		JobCh:  make(chan string, cap),
		locker: sync.RWMutex{},
	}
}

func (m *JobManager) RegisterJob(arid, jobType string, totalNodes int64) (err error) {
	if m.exist(arid) {
		return errors.New("exist job")
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	if len(m.JobCh) >= m.cap {
		err = fmt.Errorf("fully loaded")
		return
	}

	m.status[arid] = &JobStatus{
		ArId:       arid,
		JobType:    jobType,
		TotalNodes: totalNodes,
	}
	m.JobCh <- arid
	return
}

func (m *JobManager) exist(arid string) bool {
	_, ok := m.status[arid]
	return ok
}

func (m *JobManager) IncSuccessed(id string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if s, ok := m.status[id]; ok {
		s.CountSuccessed += 1
	}
}

func (m *JobManager) IncFailed(id string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if s, ok := m.status[id]; ok {
		s.CountFailed += 1
	}
}

func (m *JobManager) IncProcessed(id string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if s, ok := m.status[id]; ok {
		s.NumOfProcessed += 1
	}
}

func (m *JobManager) UnregisterJob(id string) {
	delete(m.status, id)
}

func (m *JobManager) GetJob(id string) JobStatus {
	m.locker.RLock()
	defer m.locker.RUnlock()
	job := JobStatus{}
	j, ok := m.status[id]
	if ok {
		job = *j
	}
	return job
}

func (m *JobManager) CloseJob(arId string) {
	m.locker.RLock()
	defer m.locker.RUnlock()
	job, ok := m.status[arId]
	if ok {
		job.Close = true
	}
}
func (m *JobManager) IsClosed(arId string) bool {
	job, ok := m.status[arId]
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
