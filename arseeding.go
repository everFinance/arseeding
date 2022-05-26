package arseeding

import (
	"github.com/everFinance/goar"
	"github.com/everFinance/goar/types"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"math/big"
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
	wdb         *Wdb
	bundler     *goar.Wallet
	arInfo      types.NetworkInfo
	symbolToFee map[string]*big.Int // key: tokenSymbol, val: fee per chunk_size(256KB)
	paymentExpiredRange int64 // default 1 hour
	expectedRange       int64 // default 50 block
}

func New(boltDirPath, dsn string, arWalletKeyPath string, arNode string ) *Arseeding {
	log.Debug("start new server...")
	boltDb, err := NewStore(boltDirPath)
	if err != nil {
		panic(err)
	}

	arCli := goar.NewClient("https://arweave.net")
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
	bundler, err := goar.NewWalletFromPath(arWalletKeyPath,arNode)
	if err != nil {
		panic(err)
	}

	return &Arseeding{
		store:               boltDb,
		engine:              gin.Default(),
		submitLocker:        sync.Mutex{},
		endOffsetLocker:     sync.Mutex{},
		arCli:               arCli,
		peers:               peers,
		jobManager:          jobmg,
		scheduler:           gocron.NewScheduler(time.UTC),
		wdb:                 wdb,
		bundler:             bundler,
		arInfo:              types.NetworkInfo{},
		symbolToFee:         nil,
		paymentExpiredRange: int64(60 * time.Minute),
		expectedRange:       50,
	}
}

func (s *Arseeding) Run(port string) {
	go s.runAPI(port)
	go s.runJobs()
	go s.BroadcastSubmitTx()
}
