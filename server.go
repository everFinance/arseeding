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

func New() *Server {
	boltDb, err := NewStore()
	if err != nil {
		panic(err)
	}

	arCli := goar.NewClient("https://arweave.net")
	peers, err := arCli.GetPeers()
	if err != nil {
		panic(err)
	}

	return &Server{
		store:           boltDb,
		engine:          gin.Default(),
		submitLocker:    sync.Mutex{},
		endOffsetLocker: sync.Mutex{},

		arCli:      arCli,
		peers:      peers,
		jobManager: NewJobManager(200),
		scheduler:  gocron.NewScheduler(time.UTC),
	}
}

func (s *Server) Run(port string) {
	go s.runAPI(port)
	go s.runJobs()
}
