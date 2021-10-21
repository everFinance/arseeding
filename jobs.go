package seeding

import "fmt"

const (
	jobTypeBroadcast = "broadcast"
	jobTypeSync      = "sync"
)

type JobState struct {
	Id             string `json:"id"`
	ArId           string `json:"arid"`
	CountSuccessed int64  `json:"count_successed"`
	CountFailed    int64  `json:"count_failed"`
	TotalNodes     int64  `json:"total_nodes"`
	NumOfProcessed int64  `json:"num_processed"`
}

func (s *Server) startJobs(jobType, arid string) (err error) {
	switch jobType {
	case jobTypeBroadcast:
		err = s.startBroadcastJob(arid)
	case jobTypeSync:
		err = s.startSyncJob(arid)
	default:
		err = fmt.Errorf("invalid jobType:%s", jobType)
	}

	return
}

func (s *Server) startBroadcastJob(arid string) (err error) {
	if !s.store.IsExist(arid) {
		return fmt.Errorf("not found")
	}

	go s.broadcastJob(arid)
	return
}

func (s *Server) startSyncJob(arid string) (err error) {
	// TODO check arid is exists

	go s.syncJob(arid)
	return
}

func (s *Server) broadcastJob(arid string) {}

func (s *Server) syncJob(arid string) {}
