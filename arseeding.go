package arseeding

import (
	"encoding/json"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/everpay/sdk"
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"sync"
	"time"
)

var log = NewLog("arseeding")

type Arseeding struct {
	store           *Store
	engine          *gin.Engine
	submitLocker    sync.Mutex
	endOffsetLocker sync.Mutex

	arCli     *goar.Client
	taskMg    *TaskManager
	scheduler *gocron.Scheduler

	cache *Cache

	// ANS-104
	paySdk              *sdk.SDK
	wdb                 *Wdb
	bundler             *goar.Wallet
	arInfo              types.NetworkInfo
	bundlePerFeeMap     map[string]schema.Fee // key: tokenSymbol, val: fee per chunk_size(256KB)
	paymentExpiredRange int64                 // default 1 hour
	expectedRange       int64                 // default 50 block
}

func New(boltDirPath, dsn string, arWalletKeyPath string, arNode, payUrl string) *Arseeding {
	boltDb, err := NewStore(boltDirPath)
	if err != nil {
		panic(err)
	}

	jobmg := NewTaskMg()
	if err := jobmg.InitTaskMg(boltDb); err != nil {
		panic(err)
	}

	wdb := NewWdb(dsn)
	if err = wdb.Migrate(); err != nil {
		panic(err)
	}
	bundler, err := goar.NewWalletFromPath(arWalletKeyPath, arNode)
	if err != nil {
		panic(err)
	}

	paySdk, err := sdk.New(bundler.Signer, payUrl)
	if err != nil {
		panic(err)
	}

	arCli := goar.NewClient(arNode)
	a := &Arseeding{
		store:               boltDb,
		engine:              gin.Default(),
		submitLocker:        sync.Mutex{},
		endOffsetLocker:     sync.Mutex{},
		arCli:               arCli,
		taskMg:              jobmg,
		scheduler:           gocron.NewScheduler(time.UTC),
		paySdk:              paySdk,
		wdb:                 wdb,
		bundler:             bundler,
		arInfo:              types.NetworkInfo{},
		bundlePerFeeMap:     make(map[string]schema.Fee),
		paymentExpiredRange: int64(3600),
		expectedRange:       50,
	}

	// init cache
	peerMap, err := boltDb.LoadPeers()
	if err != nil {
		log.Warn("not available peer local")
		peerMap = make(map[string]int64)
	}
	c := &Cache{peerMap: peerMap}
	peers := c.GetPeers()
	arInfo, err := fetchArInfo(arCli, peers)
	if err != nil {
		panic(err)
	}
	c.UpdateInfo(arInfo)

	fee, err := fetchArFee(arCli, peers)
	if err != nil {
		panic(err)
	}
	c.UpdateFee(fee)

	anchor, err := fetchAnchor(arCli, peers)
	if err != nil {
		panic(err)
	}
	c.UpdateAnchor(anchor)

	constTx := &types.Transaction{}
	if err := json.Unmarshal([]byte(schema.ConstTx), constTx); err != nil {
		panic(err)
	}
	c.constTx = constTx
	a.cache = c
	return a
}

func (s *Arseeding) Run(port string) {
	go s.runAPI(port)
	go s.runJobs()
	go s.runTask()
}
