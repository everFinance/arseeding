package arweave_pool_broadcast

import (
	"github.com/everFinance/arseeding"
	"github.com/everFinance/goar"
	"github.com/go-co-op/gocron"
	"gopkg.in/h2non/gentleman.v2"
	"sync"
	"time"
)

var log = arseeding.NewLog("arweave_pool_broadcast")

type BcPool struct {
	arCli     *goar.Client
	seedCli   *gentleman.Client
	scheduler *gocron.Scheduler

	pendingTxMap map[string]struct{} // key: arId, val: {}
	syncMap      map[string]bool     // key: arId, val: whether synced
	broadcastMap map[string]bool     // key: arId, val: finished is true

	mapLock sync.RWMutex
}

func New(seedUrl string) *BcPool {
	return &BcPool{
		arCli:        goar.NewClient("https://arweave.net"),
		seedCli:      gentleman.New().URL(seedUrl),
		scheduler:    gocron.NewScheduler(time.UTC),
		pendingTxMap: make(map[string]struct{}),
		syncMap:      make(map[string]bool),
		broadcastMap: make(map[string]bool),
		mapLock:      sync.RWMutex{},
	}
}

func (b *BcPool) Run() {
	go b.runJobs()
}
