package arseeding

func (s *Server) SyncTx() {
	for {
		select {
		case arId := <-s.jobManager.PopSyncTxChan():
			go func(arId string) {
				if err := s.processSyncJob(arId); err != nil {
					log.Error("s.processSyncTxJob(arId)", "err", err, "arId", arId)
				} else {
					log.Debug("success processSyncTxJob", "arId", arId)
				}
				if err := s.setProcessedJobs([]string{arId}, jobTypeSync); err != nil {
					log.Error("s.setProcessedJobs(arId)", "err", err, "arId", arId)
				}
			}(arId)
		}
	}

}

func (s *Server) syncTx(arid string) (err error) {
	if err = s.jobManager.RegisterJob(arid, jobTypeSync); err != nil {
		log.Error("Register fail", "err", err)
		return
	}

	if err = s.store.PutPendingPool(jobTypeSync, arid); err != nil {
		s.jobManager.UnregisterJob(arid, jobTypeSync)
		log.Error("PutPendingPool(jobTypeSync, arTx.ID)", "err", err, "arId", arid)
		return
	}
	s.jobManager.PutToSyncTxChan(arid)
	return nil
}
