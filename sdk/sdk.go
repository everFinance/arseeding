package sdk

import (
	"errors"
	"fmt"
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
	req := a.SCli.Post()
	req.AddPath(fmt.Sprintf("/job/broadcast/%s", arId))
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
