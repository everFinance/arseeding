package arseeding

import (
	"github.com/everFinance/goar"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"sync"
	"time"
)

var log = NewLog("arseeding")

type Server struct {
	store           *Store
	engine          *gin.Engine
	submitLocker    sync.Mutex
	endOffsetLocker sync.Mutex

	arCli      *goar.Client
	peers      []string
	jobManager *JobManager
	scheduler  *gocron.Scheduler
}

func New(boltDirPath string) *Server {
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

	return &Server{
		store:           boltDb,
		engine:          gin.Default(),
		submitLocker:    sync.Mutex{},
		endOffsetLocker: sync.Mutex{},

		arCli:      arCli,
		peers:      peers,
		jobManager: jobmg,
		scheduler:  gocron.NewScheduler(time.UTC),
	}
}

func (s *Server) Run(port string) {
	go s.runAPI(port)
	go s.runJobs()
	go s.BroadcastSubmitTx()
}
