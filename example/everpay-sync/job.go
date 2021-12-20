package everpay_sync

import "github.com/everFinance/arseeding/example"

func (e *EverPaySync) runJobs() {
	e.scheduler.Every(30).Seconds().SingletonMode().Do(e.FetchArIds)
	e.scheduler.Every(30).Seconds().SingletonMode().Do(e.PostToArseeding)

	e.scheduler.StartAsync()
}

func (e *EverPaySync) FetchArIds() {
	processedArTx, err := e.wdb.GetLastPostedTx()
	if err != nil {
		panic(err)
	}

	arIds, err := fetchTxIds(e.rollupOwner, processedArTx.ArId, e.arCli)
	if err != nil {
		panic(err)
	}
	arIds = reverseIDs(arIds)

	rollupTxIds := make([]*RollupArId, 0, len(arIds))
	for _, arId := range arIds {
		rollupTxIds = append(rollupTxIds, &RollupArId{ArId: arId})
	}
	if err := e.wdb.Insert(rollupTxIds); err != nil {
		panic(err)
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
