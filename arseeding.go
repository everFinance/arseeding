package arseeding

import (
	"github.com/everFinance/arseeding/config"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/arseeding/sdk"
	"github.com/everFinance/everpay-go/common"
	paySdk "github.com/everFinance/everpay-go/sdk"
	"github.com/everFinance/goar"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"sync"
	"time"
)

var log = common.NewLog("arseeding")

type Arseeding struct {
	store           *Store
	engine          *gin.Engine
	submitLocker    sync.Mutex
	endOffsetLocker sync.Mutex

	arCli     *goar.Client
	taskMg    *TaskManager
	scheduler *gocron.Scheduler

	cache  *Cache
	config *config.Config

	// ANS-104 bundle
	arseedCli           *sdk.ArSeedCli
	everpaySdk          *paySdk.SDK
	wdb                 *Wdb
	bundler             *goar.Wallet
	NoFee               bool                  // if true, means no bundle fee; default false
	bundlePerFeeMap     map[string]schema.Fee // key: tokenSymbol, val: fee per chunk_size(256KB)
	paymentExpiredRange int64                 // default 1 hour
	expectedRange       int64                 // default 50 block
}

func New(
	boltDirPath, dsn string,
	arWalletKeyPath string, arNode, payUrl string, noFee bool,
	useS3 bool, s3AccKey, s3SecretKey, s3BucketPrefix, s3Region string, use4EVER bool,
	useS3 bool, s3AccKey, s3SecretKey, s3BucketPrefix, s3Region string, port string,
) *Arseeding {
	var err error
	KVDb := &Store{}
	if useS3 {
		KVDb, err = NewS3Store(s3AccKey, s3SecretKey, s3Region, s3BucketPrefix, use4EVER)
	} else {
		KVDb, err = NewBoltStore(boltDirPath)
	}
	if err != nil {
		panic(err)
	}

	jobmg := NewTaskMg()
	if err := jobmg.InitTaskMg(KVDb); err != nil {
		panic(err)
	}

	wdb := NewWdb(dsn)
	if err = wdb.Migrate(noFee); err != nil {
		panic(err)
	}
	localArseedUrl := "http://127.0.0.1" + port
	bundler, err := goar.NewWalletFromPath(arWalletKeyPath, localArseedUrl)
	if err != nil {
		panic(err)
	}

	everpaySdk, err := paySdk.New(bundler.Signer, payUrl)
	if err != nil {
		panic(err)
	}

	arCli := goar.NewClient(arNode)
	a := &Arseeding{
		config:              config.New(dsn),
		store:               KVDb,
		engine:              gin.Default(),
		submitLocker:        sync.Mutex{},
		endOffsetLocker:     sync.Mutex{},
		arCli:               arCli,
		taskMg:              jobmg,
		scheduler:           gocron.NewScheduler(time.UTC),
		arseedCli:           sdk.New(localArseedUrl),
		everpaySdk:          everpaySdk,
		wdb:                 wdb,
		bundler:             bundler,
		NoFee:               noFee,
		bundlePerFeeMap:     make(map[string]schema.Fee),
		paymentExpiredRange: schema.DefaultPaymentExpiredRange,
		expectedRange:       schema.DefaultExpectedRange,
	}

	// init cache
	peerMap, err := KVDb.LoadPeers()
	if err != nil {
		peerMap = make(map[string]int64)
	}
	a.cache = NewCache(arCli, peerMap)
	return a
}

func (s *Arseeding) Run(port string) {
	s.config.Run()
	go s.runAPI(port)
	go s.runJobs()
	go s.runTask()
}
