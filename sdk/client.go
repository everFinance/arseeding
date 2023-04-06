package sdk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"gopkg.in/h2non/gentleman.v2"
	"io"
	"strconv"
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

func (a *ArSeedCli) SubmitTxConcurrent(ctx context.Context, concurrentNum int, arTx types.Transaction) error {
	uploader, err := goar.CreateUploader(a.ACli, &arTx, nil)
	if err != nil {
		return err
	}
	return uploader.ConcurrentOnce(ctx, concurrentNum)
}

func (a *ArSeedCli) BroadcastTxData(arId string) error {
	return a.postTask(schema.TaskTypeBroadcast, arId)
}

func (a *ArSeedCli) BroadcastTxMeta(arId string) error {
	return a.postTask(schema.TaskTypeBroadcastMeta, arId)
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
	addr := schema.ResBundler{}
	err = resp.JSON(&addr)
	return addr.Bundler, err
}

func (a *ArSeedCli) SubmitItem(itemBinary []byte, currency string, apikey string, needSequence bool) (*schema.RespOrder, error) {
	req := a.SCli.Post()
	if currency != "" {
		req.Path(fmt.Sprintf("/bundle/tx/%s", currency))
	} else {
		req.Path("/bundle/tx")
	}

	req.SetHeader("Content-Type", "application/octet-stream")
	if len(apikey) > 0 {
		req.SetHeader("X-API-KEY", apikey)
	}
	if needSequence {
		req.SetHeader("Sort", "true")
	}

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

func (a *ArSeedCli) SubmitItemStream(itemBinary io.Reader, currency string, apikey string, needSequence bool) (*schema.RespOrder, error) {
	req := a.SCli.Post()
	req.Path(fmt.Sprintf("/bundle/tx/%s", currency))
	req.SetHeader("Content-Type", "application/octet-stream")
	if len(apikey) > 0 {
		req.SetHeader("X-API-KEY", apikey)
	}
	if needSequence {
		req.SetHeader("Sort", "true")
	}

	req.Body(itemBinary)

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

func (a *ArSeedCli) SubmitNativeData(apiKey string, currency string, data []byte, contentType string, tags map[string]string) (*schema.RespItemId, error) {
	req := a.SCli.Post()
	req.Path(fmt.Sprintf("/bundle/data/%s", currency))
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

func (a *ArSeedCli) SubmitNativeDataStream(apiKey string, currency string, data io.Reader, contentType string, tags map[string]string) (*schema.RespItemId, error) {
	req := a.SCli.Post()
	req.Path(fmt.Sprintf("/bundle/data/%s", currency))
	req.SetHeader("X-API-KEY", apiKey)
	req.AddQuery("Content-Type", contentType)
	for k, v := range tags {
		req.AddQuery(k, v)
	}
	req.Body(data)

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

func (a *ArSeedCli) GetItemData(itemId string) ([]byte, error) {
	req := a.SCli.Get()
	req.Path(fmt.Sprintf("/%s", itemId))

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	if !resp.Ok {
		return nil, errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}
	return resp.Bytes(), err
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

func (a *ArSeedCli) GetOrders(addr string, startId int) ([]schema.Order, error) {
	req := a.SCli.Get()
	req.Path(fmt.Sprintf("/bundle/orders/%s", addr))
	req.AddQuery("cursorId", strconv.Itoa(startId))
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

func (a *ArSeedCli) GetApiKey(addr string) (schema.RespApiKey, error) {
	req := a.SCli.Get()
	req.Path(fmt.Sprintf("/apikey/%s", addr))
	resp, err := req.Send()
	if err != nil {
		return schema.RespApiKey{}, err
	}
	defer resp.Close()
	if !resp.Ok {
		return schema.RespApiKey{}, errors.New(fmt.Sprintf("resp failed: %s", resp.String()))
	}

	apiKey := schema.RespApiKey{}
	err = resp.JSON(&apiKey)
	return apiKey, err
}
