package everpay

import (
	"errors"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"gopkg.in/h2non/gentleman.v2"
	"sync"
	"time"
)

func (e *EverPay) runJobs() {
	e.scheduler.Every(30).Seconds().SingletonMode().Do(e.FetchArIds)
	e.scheduler.Every(30).Seconds().SingletonMode().Do(e.PostToArseeding)

	e.scheduler.StartAsync()
}

func (e *EverPay) FetchArIds() {
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

func (e *EverPay) PostToArseeding() {
	rollupTxs, err := e.wdb.GetNeedPostTxs()
	if err != nil {
		panic(err)
	}
	if len(rollupTxs) == 0 {
		log.Debug("no need post seeding server")
		return
	}

	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(20, func(i interface{}) {
		defer wg.Done()
		arId := i.(string)
		if err := postSyncJob(arId, e.gtmCli); err != nil {
			log.Error("postSyncJob(rtx.ArId)", "err", err, "arId", arId)
			if err.Error() == "\"fully loaded\"" {
				log.Debug("retry", "arId", arId)
				for {
					time.Sleep(1 * time.Second)
					if err := postSyncJob(arId, e.gtmCli); err == nil {
						return
					}
				}
			}
			if err.Error() != "\"arId has successed synced\"" {
				return
			}
		}
		// update post status is true
		if err := e.wdb.UpdatePosted(arId); err != nil {
			log.Error("e.wdb.UpdatePosted(arId)", "err", err, "arId", arId)
		}
	})

	defer p.Release()
	for _, rtx := range rollupTxs {
		wg.Add(1)
		_ = p.Invoke(rtx.ArId)
	}
	wg.Wait()
}

func postSyncJob(arId string, cli *gentleman.Client) error {
	req := cli.Request()
	req.AddPath(fmt.Sprintf("/job/sync/%s", arId))
	req.Method("POST")
	resp, err := req.Send()
	if err != nil {
		return err
	}
	if !resp.Ok {
		return errors.New(resp.String())
	}
	return nil
}

func reverseIDs(ids []string) (rIDs []string) {
	for i := len(ids) - 1; i >= 0; i-- {
		rIDs = append(rIDs, ids[i])
	}
	return
}
