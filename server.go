package seeding

import (
	"github.com/gin-gonic/gin"
	"sync"
)

var log = NewLog("seeding")

type Server struct {
	store        *Store
	engine       *gin.Engine
	submitLocker sync.Mutex
}

func New() *Server {
	boltDb, err := NewStore()
	if err != nil {
		panic(err)
	}
	return &Server{
		store:        boltDb,
		engine:       gin.Default(),
		submitLocker: sync.Mutex{},
	}
}

func (s *Server) Run(port string) {
	go s.runAPI(port)
}
