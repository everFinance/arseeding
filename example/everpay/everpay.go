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

// 	dsn := "root@tcp(127.0.0.1:3306)/sandy_test?charset=utf8mb4&parseTime=True&loc=Local"
// seedUrl := "https://seed-dev.everpay.io"
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
