package arweave_pool_broadcast

import "github.com/everFinance/arseeding/example"

func (b *BcPool) runJobs() {
	b.scheduler.Every(30).Seconds().SingletonMode().Do(b.updatePendingTxIds)
	b.scheduler.Every(30).Seconds().SingletonMode().Do(b.syncPendingTx)
	b.scheduler.Every(1).Minutes().SingletonMode().Do(b.broadcastPendingTx)
	b.scheduler.Every(2).Minutes().SingletonMode().Do(b.checkBroadcastTxStatus)

	b.scheduler.StartAsync()
}

func (b *BcPool) syncPendingTx() {
	unsyncedTxIds := make([]string, 0, 100)
	b.mapLock.RLock()
	for txId, synced := range b.syncMap {
		if !synced {
			unsyncedTxIds = append(unsyncedTxIds, txId)
		}
	}
	b.mapLock.RUnlock()

	if len(unsyncedTxIds) == 0 {
		return
	}
	// sync
	successTxIds := example.MustBatchSyncTxIds(unsyncedTxIds, b.seedCli)
	// update
	b.mapLock.Lock()
	for _, txId := range successTxIds {
		b.syncMap[txId] = true
	}
	b.mapLock.Unlock()
	log.Debug("sync pending tx success...", "num", len(successTxIds))
}

func (b *BcPool) broadcastPendingTx() {
	b.mapLock.RLock()
	syncedTxIds := make([]string, 0, 100)
	for txId, synced := range b.syncMap {
		if synced {
			syncedTxIds = append(syncedTxIds, txId)
		}
	}

	// filter has broadcast
	needBroadcastTxIds := make([]string, 0, len(syncedTxIds))
	for _, txId := range syncedTxIds {
		if _, ok := b.broadcastMap[txId]; ok {
			continue
		}
		needBroadcastTxIds = append(needBroadcastTxIds, txId)
	}
	b.mapLock.RUnlock()

	if len(needBroadcastTxIds) == 0 {
		return
	}
	// broadcast
	successTxIds := example.MustBatchBroadcastTxIds(needBroadcastTxIds, b.seedCli)
	// update
	b.mapLock.Lock()
	for _, txId := range successTxIds {
		b.broadcastMap[txId] = false
	}
	b.mapLock.Unlock()
	log.Debug("broadcast tx success...", "num", len(successTxIds))
}

func (b *BcPool) updatePendingTxIds() {
	pendingTxIds, err := b.arCli.GetPendingTxIds()
	if err != nil {
		log.Error("b.arCli.GetPendingTxIds()", "err", err)
		return
	}
	b.mapLock.Lock()
	defer b.mapLock.Unlock()

	// insert maps
	count := 0
	for _, txId := range pendingTxIds {
		if _, ok := b.pendingTxMap[txId]; ok {
			continue
		}
		b.pendingTxMap[txId] = struct{}{}
		b.syncMap[txId] = false
		count++
	}
	if count > 0 {
		log.Debug("update pending tx map success...", "num", count)
	}
}

func (b *BcPool) checkBroadcastTxStatus() {
	needCheckBroadcastTxIds := make([]string, 0)
	b.mapLock.RLock()
	for txId, finished := range b.broadcastMap {
		if !finished {
			needCheckBroadcastTxIds = append(needCheckBroadcastTxIds, txId)
		}
	}
	b.mapLock.Unlock()

	if len(needCheckBroadcastTxIds) == 0 {
		return
	}

	for _, arId := range needCheckBroadcastTxIds {
		tkStatus, err := b.seedCli.GetBroadcastTask(arId)
		if err != nil {
			log.Error("b.seedCli.GetBroadcastTask(arId)", "err", err, "arId", arId)
			continue
		}
		if tkStatus.CountSuccessed >= 10 {
			// close job
			if err := b.seedCli.KillBroadcastTask(arId); err != nil {
				log.Error("b.seedCli.KillBroadcastTask(arId)", "err", err, "arId", arId)
				continue
			}

			// update broadcast finish
			b.broadcastMap[arId] = true
		}
	}
	log.Debug("checkBroadcastTxStatus success...", "num", len(needCheckBroadcastTxIds))
}
