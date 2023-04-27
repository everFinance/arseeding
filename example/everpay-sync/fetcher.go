package everpay_sync

import (
	"errors"
	"fmt"
	"github.com/everFinance/go-everpay/common"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"time"
)

var (
	ErrNotNeedSync = errors.New("not need sync")
	log            = common.NewLog("example_everpay")
)

func (e *EverPaySync) fetchTxIds(processedArTxId string) (err error) {
	// get ar tx
	// processArParentId, err := getParentIdByTags(processedArTxId, e.arCli)
	// if err != nil {
	// 	return  err
	// }

	// arOwnerLastTxId, err := getLastTxId(e.rollupOwner, processedArTxId, processArParentId, e.arCli)
	// if err != nil {
	// 	if err == ErrNotNeedSync {
	// 		return  nil
	// 	}
	// 	log.Warn("getLastTxId(arOwner, processedArTxId, c)", "err", err)
	// 	return  err
	// }

	if processedArTxId == "" {
		processedArTxId, err = e.arCli.GetLastTransactionID(e.rollupOwner)
		if err != nil {
			return err
		}
	}
	var (
		parentTxId string
	)
	id := processedArTxId
	for {
		parentTxId, err = getParentIdByTags(id, e.arCli)
		if err != nil {
			return err
		}
		// if parentTxId == processedArTxId || parentTxId == processArParentId {
		// 	break
		// }
		if parentTxId == "" {
			break
		}
		id = parentTxId
		e.arIdChan <- parentTxId
	}
	return nil
}

func getLastTxId(arOwner string, processedArTxId string, processedParentArTxId string, c *goar.Client) (string, error) {
	var (
		lastTxId string
		err      error
	)
	// 1. get owner last tx id
	lastTxId, err = c.GetLastTransactionID(arOwner)
	if err != nil {
		log.Error("c.GetLastTransactionID(arOwner)", "err", err)
		return "", err
	}
	log.Warn("get processedArTxId", "tracker processedArTxId", processedArTxId, "current lastTxId", lastTxId)

	if len(lastTxId) == 0 {
		return "", errors.New("get arOwner last txId nil")
	}

	// 2. check lastTxId is packaged ar tx
	// 2.1 if same id, not need sync
	if lastTxId == processedArTxId {
		log.Debug("lastTxId == processedArTxId", "do not need sync rollup tx, processedArTxId", processedArTxId, "lastTxId", lastTxId)
		return "", ErrNotNeedSync
	}

	// check lastTxId must more than 2 block confirms
	lastTxStatus, err := c.GetTransactionStatus(lastTxId)
	if err != nil {
		log.Error("native-fetch c.GetTransactionStatus(lastTxId)", "err", err, "lastTxId", lastTxId)
		return "", err
	}
	// if lastTx confirms block < 3, so we need use parentTxId as lastTxId
	if lastTxStatus.NumberOfConfirmations < 3 {
		// get parent txId as lastTxId
		lastTxId, err = getParentIdByTags(lastTxId, c)
		if err != nil {
			log.Error("get lastTx parentId tags", "err", err, "lastTxId", lastTxId)
			return "", err
		}
		// lastTxId must > processedArTxId,so can not == processedArTxId || processedParentArTxId
		if lastTxId == processedArTxId || lastTxId == processedParentArTxId {
			log.Debug("lastTxId == processedArTxId ||  lastTxId == processedParentArTxId; do not need sync rollup tx", "processedArTxId", processedArTxId, "lastTxId", lastTxId, "processedParentArTxId", processedParentArTxId)
			return "", ErrNotNeedSync
		}
	}

	// 2.2 lastTxId can not less than processedArTxId
	if len(processedArTxId) != 0 {
		lastTxStatus, err = c.GetTransactionStatus(lastTxId)
		if err != nil {
			log.Error("c.GetTransactionStatus(lastTxId)", "err", err, "lastTxId", lastTxId)
			return "", err
		}
		processedArTxStatus, err := c.GetTransactionStatus(processedArTxId)
		if err != nil {
			log.Error("c.GetTransactionStatus(processedArTxId)", "err", err, "processedArTxId", processedArTxId)
			return "", err
		}
		// lastTxHeight must >= processedArTxHeight
		if lastTxStatus.BlockHeight < processedArTxStatus.BlockHeight {
			return "", fmt.Errorf("lastTxId must more than processedArTxId; lastTxId: %s,processedArTxId: %s", lastTxId, processedArTxId)
		}
	}
	return lastTxId, nil
}

func getParentIdByTags(arId string, c *goar.Client) (string, error) {
	if arId == "" {
		return "", nil
	}
	var (
		tags []types.Tag
		err  error
	)
	for {
		tags, err = c.GetTransactionTags(arId)
		if err == nil || err != goar.ErrBadGateway {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return "", err
	}
	// get parent id
	mapTags := make(map[string]string)
	for _, tag := range tags {
		mapTags[tag.Name] = tag.Value
	}
	if parentId, ok := mapTags["parent_id"]; ok {
		return parentId, nil
	}
	return "", errors.New("get rollup tx tags nil")
}
