package everpay_sync

import (
	"github.com/everFinance/arseeding/sdk"
	"github.com/everFinance/goar"
	"github.com/go-co-op/gocron"
	"time"
)

// function: sync everpay rollup txs to arseeding
type EverPaySync struct {
	wdb         *Wdb
	arCli       *goar.Client
	rollupOwner string
	gtmCli      *sdk.ArSeedCli

	scheduler *gocron.Scheduler
	arIdChan  chan string
}

func New(dsn string, seedUrl string) *EverPaySync {
	return &EverPaySync{
		wdb:         NewWdb(dsn),
		arCli:       goar.NewClient("https://arweave.net"),
		rollupOwner: "uGx-QfBXSwABKxjha-00dI7vvfyqIYblY6Z5L6cyTFM",
		gtmCli:      sdk.New(seedUrl),
		scheduler:   gocron.NewScheduler(time.UTC),
		arIdChan:    make(chan string),
	}
}

func (e *EverPaySync) Run() {
	go e.runJobs()
	go e.FetchArIds()
}
