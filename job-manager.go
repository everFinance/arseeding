package seeding

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

const (
	jobTypeBroadcast = "broadcast"
	jobTypeSync      = "sync"
)

type JobStatus struct {
	Id             string `json:"id"`
	ArId           string `json:"arid"`
	JobType        string `json:"jobType"`
	CountSuccessed int64  `json:"countSuccessed"`
	CountFailed    int64  `json:"countFailed"`
	TotalNodes     int64  `json:"totalNodes"`
	NumOfProcessed int64  `json:"numProcessed"`
}

type JobManager struct {
	cap    int64
	status map[string]*JobStatus
	locker sync.RWMutex
}

func NewJobManager(cap int64) *JobManager {
	return &JobManager{cap: cap, status: make(map[string]*JobStatus)}
}

func (m *JobManager) RegisterJob(arid, jobType string, totalNodes int64) (id string, err error) {
	id = uuid.NewString()

	m.locker.Lock()
	defer m.locker.Unlock()

	if int64(len(m.status)) >= m.cap {
		id = ""
		err = fmt.Errorf("fully loaded")
		return
	}

	m.status[id] = &JobStatus{
		Id:         id,
		ArId:       arid,
		JobType:    jobType,
		TotalNodes: totalNodes,
	}
	return
}

func (m *JobManager) IncSuccessed(id string) {
	m.locker.Lock()
	defer m.locker.Unlock()

	if s, ok := m.status[id]; ok {
		s.CountSuccessed += 1
	}
}

func (m *JobManager) IncFalied(id string) {
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

func (m *JobManager) RemoveJob(id string) {
	delete(m.status, id)
}

// func (m *JobManager) GetJob(Id string) {}

func (m *JobManager) GetJobs() (jobs map[string]JobStatus) {
	m.locker.RLock()
	defer m.locker.RUnlock()

	jobs = make(map[string]JobStatus, len(m.status))
	for id, job := range m.status {
		jobs[id] = *job
	}
	return
}
