package everpay

import (
	"github.com/everFinance/goar"
	"github.com/go-co-op/gocron"
	"gopkg.in/h2non/gentleman.v2"
	"time"
)

type EverPay struct {
	wdb         *Wdb
	arCli       *goar.Client
	rollupOwner string
	gtmCli      *gentleman.Client

	scheduler *gocron.Scheduler
}

func New(dsn string, seedUrl string) *EverPay {
	return &EverPay{
		wdb:         NewWdb(dsn),
		arCli:       goar.NewClient("https://arweave.net"),
		rollupOwner: "uGx-QfBXSwABKxjha-00dI7vvfyqIYblY6Z5L6cyTFM",
		gtmCli:      gentleman.New().URL(seedUrl),
		scheduler:   gocron.NewScheduler(time.UTC),
	}
}

func (e *EverPay) Run() {
	go e.runJobs()
}
