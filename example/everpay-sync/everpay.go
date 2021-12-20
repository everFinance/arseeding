package everpay_sync

import (
	"github.com/everFinance/goar"
	"github.com/go-co-op/gocron"
	"gopkg.in/h2non/gentleman.v2"
	"time"
)

// function: sync everpay rollup txs to arseeding
type EverPaySync struct {
	wdb         *Wdb
	arCli       *goar.Client
	rollupOwner string
	gtmCli      *gentleman.Client

	scheduler *gocron.Scheduler
}

func New(dsn string, seedUrl string) *EverPaySync {
	return &EverPaySync{
		wdb:         NewWdb(dsn),
		arCli:       goar.NewClient("https://arweave.net"),
		rollupOwner: "uGx-QfBXSwABKxjha-00dI7vvfyqIYblY6Z5L6cyTFM",
		gtmCli:      gentleman.New().URL(seedUrl),
		scheduler:   gocron.NewScheduler(time.UTC),
	}
}

func (e *EverPaySync) Run() {
	go e.runJobs()
}
