package everpay_sync

import "github.com/everFinance/arseeding/example"

func (e *EverPaySync) runJobs() {
	e.scheduler.Every(30).Seconds().SingletonMode().Do(e.PostToArseeding)

	e.scheduler.StartAsync()
}

func (e *EverPaySync) FetchArIds() {
	go func(e *EverPaySync) {
		processedArTx, err := e.wdb.GetLastPostTx()
		if err != nil {
			panic(err)
		}
		err = e.fetchTxIds(processedArTx.ArId)
		if err != nil {
			panic(err)
		}
	}(e)

	for {
		select {
		case arId := <-e.arIdChan:
			if err := e.wdb.Insert(arId); err != nil {
				panic(err)
			}
		}
	}
}

func (e *EverPaySync) PostToArseeding() {
	rollupTxs, err := e.wdb.GetNeedPostTxs()
	if err != nil {
		panic(err)
	}
	if len(rollupTxs) == 0 {
		log.Debug("no need post seeding server")
		return
	}
	txIds := make([]string, 0, len(rollupTxs))
	for _, arTx := range rollupTxs {
		txIds = append(txIds, arTx.ArId)
	}

	successTxIds := example.MustBatchSyncTxIds(txIds, e.gtmCli)

	// update db
	for _, arId := range successTxIds {
		// update post status is true
		if err := e.wdb.UpdatePosted(arId); err != nil {
			log.Error("e.wdb.UpdatePosted(arId)", "err", err, "arId", arId)
		}
	}
}

func reverseIDs(ids []string) (rIDs []string) {
	for i := len(ids) - 1; i >= 0; i-- {
		rIDs = append(rIDs, ids[i])
	}
	return
}
