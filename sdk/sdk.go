package sdk

import (
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"gopkg.in/h2non/gentleman.v2"
)

type ArSeedCli struct {
	ACli *goar.Client
	SCli *gentleman.Client
}

func New(arSeedUrl string) *ArSeedCli {
	return &ArSeedCli{
		ACli: goar.NewClient(arSeedUrl),
		SCli: gentleman.New().URL(arSeedUrl),
	}
}

func (a *ArSeedCli) SubmitTx(arTx types.Transaction) error {
	uploader, err := goar.CreateUploader(a.ACli, &arTx, nil)
	if err != nil {
		return err
	}
	return uploader.Once()
}

func (a *ArSeedCli) BroadcastTxData(arId string) error {
	return a.postTask(schema.TaskTypeBroadcast, arId)
}

func (a *ArSeedCli) SyncTx(arId string) error {
	return a.postTask(schema.TaskTypeSync, arId)
}

func (a *ArSeedCli) GetBroadcastTask(arId string) (schema.Task, error) {
	return a.getTask(schema.TaskTypeBroadcast, arId)
}

func (a *ArSeedCli) GetBroadcastMetaTask(arId string) (schema.Task, error) {
	return a.getTask(schema.TaskTypeBroadcastMeta, arId)
}

func (a *ArSeedCli) GetSyncTask(arId string) (schema.Task, error) {
	return a.getTask(schema.TaskTypeSync, arId)
}

func (a *ArSeedCli) KillBroadcastTask(arId string) error {
	return a.killTask(schema.TaskTypeBroadcast, arId)
}

func (a *ArSeedCli) KillBroadcastMetaTask(arId string) error {
	return a.killTask(schema.TaskTypeBroadcastMeta, arId)
}

func (a *ArSeedCli) KillSyncTask(arId string) error {
	return a.killTask(schema.TaskTypeSync, arId)
}

func (a *ArSeedCli) postTask(taskType, arId string) error {
	req := a.SCli.Post()
	req.AddPath(fmt.Sprintf("/task/%s/%s", taskType, arId))
	resp, err := req.Send()
	if err != nil {
		fmt.Printf("req.Send() error: %v\n", err)
		return err
	}
	defer resp.Close()
	if !resp.Ok {
		return errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}
	return nil
}

func (a *ArSeedCli) getTask(taskType, arId string) (schema.Task, error) {
	req := a.SCli.Get()
	req.AddPath(fmt.Sprintf("/task/%s/%s", taskType, arId))
	resp, err := req.Send()
	if err != nil {
		fmt.Printf("req.Send() error: %v\n", err)
		return schema.Task{}, err
	}
	defer resp.Close()
	if !resp.Ok {
		return schema.Task{}, errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}
	tk := schema.Task{}
	err = resp.JSON(&tk)
	return tk, err
}

func (a *ArSeedCli) killTask(taskType, arId string) error {
	req := a.SCli.Post()
	req.AddPath(fmt.Sprintf("/task/kill/%s/%s", taskType, arId))
	resp, err := req.Send()
	if err != nil {
		fmt.Printf("req.Send() error: %v\n", err)
		return err
	}
	defer resp.Close()
	if !resp.Ok {
		return errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}
	return nil
}
