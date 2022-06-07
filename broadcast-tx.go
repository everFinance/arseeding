package arseeding

func (s *Server) BroadcastTx() {
	for {
		select {
		case arId := <-s.jobManager.PopBroadcastTxChan():
			go func() {
				if err := s.processBroadcastJob(arId); err != nil {
					log.Error("s.processBroadcastTxJob(arId)", "err", err, "arId", arId)
				} else {
					log.Debug("success processBroadcastTxJob", "arId", arId)
				}
				if err := s.setProcessedJobs([]string{arId}, jobTypeBroadcast); err != nil {
					log.Error("s.setProcessedJobs(arId)", "err", err, "arId", arId)
				}
			}()
		}
	}

}

func (s *Server) broadcastTx(arid string) (err error) {
	if err = s.jobManager.RegisterJob(arid, jobTypeBroadcast); err != nil {
		log.Error("Register fail", "err", err)
		return
	}

	if err = s.store.PutPendingPool(jobTypeBroadcast, arid); err != nil {
		s.jobManager.UnregisterJob(arid, jobTypeBroadcast)
		log.Error("PutPendingPool(jobTypeTxBroadcast, arTx.ID)", "err", err, "arId", arid)
		return
	}
	s.jobManager.PutToBroadcastTxChan(arid)
	return nil
}
