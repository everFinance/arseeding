package seeding

import (
	"github.com/gin-gonic/gin"
)

type Server struct {
	store *Store

	engine *gin.Engine
}

func New() *Server {
	return &Server{
		store:  &Store{},
		engine: gin.Default(),
	}
}

func (s *Server) Run(port string) {
	go s.runAPI(port)
}
