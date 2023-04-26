package example

import (
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/arseeding/sdk"
	"github.com/everFinance/go-everpay/common"
	"github.com/panjf2000/ants/v2"
	"gopkg.in/h2non/gentleman.v2"
	"sync"
	"time"
)

var log = common.NewLog("example")

func MustBatchSyncTxIds(txIds []string, seedCli *sdk.ArSeedCli) (successTxIds []string) {
	var wg sync.WaitGroup
	successTxIds = make([]string, 0, len(txIds))
	p, _ := ants.NewPoolWithFunc(20, func(i interface{}) {
		defer wg.Done()
		arId := i.(string)
		if err := seedCli.SyncTx(arId); err != nil {
			log.Error("seedCli.SyncTx(arId)", "err", err, "arId", arId)
			for {
				log.Debug("retry sync", "arId", arId)
				time.Sleep(1 * time.Second)
				if err = seedCli.SyncTx(arId); err == nil {
					successTxIds = append(successTxIds, arId)
					return
				}
			}
		}

		successTxIds = append(successTxIds, arId)
	})

	defer p.Release()
	for _, rtx := range txIds {
		wg.Add(1)
		_ = p.Invoke(rtx)
	}
	wg.Wait()
	return
}

func MustBatchBroadcastTxIds(txIds []string, seedCli *sdk.ArSeedCli) (successTxIds []string) {
	var wg sync.WaitGroup
	successTxIds = make([]string, 0, len(txIds))
	p, _ := ants.NewPoolWithFunc(20, func(i interface{}) {
		defer wg.Done()
		arId := i.(string)

		if err := seedCli.BroadcastTxData(arId); err != nil {
			log.Error("postBroadcastJob(arId)", "err", err, "arId", arId)

			for {
				log.Debug("retry", "arId", arId)
				time.Sleep(5 * time.Second)
				if err := seedCli.BroadcastTxData(arId); err == nil {
					successTxIds = append(successTxIds, arId)
					return
				}
			}

		}
		successTxIds = append(successTxIds, arId)
	})

	defer p.Release()
	for _, rtx := range txIds {
		wg.Add(1)
		_ = p.Invoke(rtx)
	}
	wg.Wait()
	return
}

func postBroadcastJob(arId string, cli *gentleman.Client) error {
	req := cli.Request()
	req.AddPath(fmt.Sprintf("/job/broadcast/%s", arId))
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

func GetJob(arId string, tktype string, cli *gentleman.Client) (*schema.Task, error) {
	req := cli.Request()
	req.AddPath(fmt.Sprintf("/job/%s/%s", arId, tktype))
	req.Method("GET")
	resp, err := req.Send()
	if err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, errors.New(resp.String())
	}
	res := &schema.Task{}
	err = resp.JSON(res)
	return res, err
}

func KillJob(arId string, tktype string, cli *gentleman.Client) error {
	req := cli.Request()
	req.AddPath(fmt.Sprintf("/job/kill/%s/%s", arId, tktype))
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
