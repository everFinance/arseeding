package sdk

import (
	"bytes"
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
		return err
	}
	defer resp.Close()
	if !resp.Ok {
		return errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}
	return nil
}

// bundle

func (a *ArSeedCli) GetBundler() (string, error) {
	req := a.SCli.Get()
	req.Path("/bundle/bundler")
	resp, err := req.Send()
	if err != nil {
		return "", err
	}
	defer resp.Close()
	if !resp.Ok {
		return "", errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}
	addr := ""
	err = resp.JSON(&addr)
	return addr, err
}

func (a *ArSeedCli) SubmitItem(itemBinary []byte, currency string) (*schema.RespOrder, error) {
	req := a.SCli.Post()
	req.Path(fmt.Sprintf("/bundle/tx/%s", currency))
	req.SetHeader("Content-Type", "application/octet-stream")

	req.Body(bytes.NewReader(itemBinary))

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if !resp.Ok {
		return nil, fmt.Errorf("send to bundler request failed; http code: %d, errMsg:%s", resp.StatusCode, resp.String())
	}
	br := &schema.RespOrder{}
	err = resp.JSON(br)
	return br, err
}

func (a *ArSeedCli) SubmitNativeData(apiKey string, data []byte, contentType string, tags map[string]string) (*schema.RespItemId, error) {
	req := a.SCli.Post()
	req.Path(fmt.Sprintf("/bundle/data"))
	req.SetHeader("X-API-KEY", apiKey)
	req.AddQuery("Content-Type", contentType)
	for k, v := range tags {
		req.AddQuery(k, v)
	}
	req.Body(bytes.NewReader(data))

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if !resp.Ok {
		return nil, fmt.Errorf("resp failed.http code: %d, errMsg:%s", resp.StatusCode, resp.String())
	}
	br := &schema.RespItemId{}
	err = resp.JSON(br)
	return br, err
}

func (a *ArSeedCli) GetItemMeta(itemId string) (types.BundleItem, error) {
	req := a.SCli.Get()
	req.Path(fmt.Sprintf("/bundle/tx/%s", itemId))

	resp, err := req.Send()
	if err != nil {
		return types.BundleItem{}, err
	}
	defer resp.Close()
	if !resp.Ok {
		return types.BundleItem{}, errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}

	item := types.BundleItem{}
	err = resp.JSON(&item)
	return item, err
}

func (a *ArSeedCli) GetItemIds(arId string) ([]string, error) {
	req := a.SCli.Get()
	req.Path(fmt.Sprintf("/bundle/itemIds/%s", arId))

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if !resp.Ok {
		return nil, errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}

	ids := make([]string, 0)
	err = resp.JSON(&ids)
	return ids, err
}

func (a *ArSeedCli) BundleFee(size int64, currency string) (schema.RespFee, error) {
	req := a.SCli.Get()
	req.Path(fmt.Sprintf("/bundle/fee/%d/%s", size, currency))

	resp, err := req.Send()
	if err != nil {
		return schema.RespFee{}, err
	}
	defer resp.Close()
	if !resp.Ok {
		return schema.RespFee{}, errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}

	fee := schema.RespFee{}
	err = resp.JSON(&fee)
	return fee, err
}

func (a *ArSeedCli) GetOrders(addr string) ([]schema.Order, error) {
	req := a.SCli.Get()
	req.Path(fmt.Sprintf("/bundle/orders/%s", addr))

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if !resp.Ok {
		return nil, errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}

	ords := make([]schema.Order, 0)
	err = resp.JSON(&ords)
	return ords, err
}
