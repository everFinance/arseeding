package arseeding

import (
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

	arCli      *goar.Client
	peers      []string
	jobManager *JobManager
	scheduler  *gocron.Scheduler

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

	arCli := goar.NewClient(arNode)
	peers, err := arCli.GetPeers()
	if err != nil {
		panic(err)
	}

	jobmg := NewJobManager(500)
	if err := jobmg.InitJobManager(boltDb); err != nil {
		panic(err)
	}

	wdb := NewWdb(dsn)
	if err := wdb.Migrate(); err != nil {
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

	a := &Arseeding{
		store:               boltDb,
		engine:              gin.Default(),
		submitLocker:        sync.Mutex{},
		endOffsetLocker:     sync.Mutex{},
		arCli:               arCli,
		peers:               peers,
		jobManager:          jobmg,
		scheduler:           gocron.NewScheduler(time.UTC),
		paySdk:              paySdk,
		wdb:                 wdb,
		bundler:             bundler,
		arInfo:              types.NetworkInfo{},
		bundlePerFeeMap:     make(map[string]schema.Fee),
		paymentExpiredRange: int64(60 * time.Minute),
		expectedRange:       50,
	}

	return a
}

func (s *Arseeding) Run(port string) {
	go s.runAPI(port)
	go s.runJobs()
	go s.BroadcastSubmitTx()
}
