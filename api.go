package seeding

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) runAPI(port string) {
	s.engine.GET("/tx/:arid", s.getTx)
	s.engine.GET("/tx/:arid/:field", s.getTxField)
	s.engine.POST("/tx", s.submitTx)

	// TODO: chunk upload and download
	// s.engine.GET("/chunk", s.getChunk)
	// s.engine.POST("/chunk", s.submitChunk)

	// broadcast & sync jobs
	s.engine.GET("/job/broadcast/:arid", s.broadcast)
	s.engine.GET("/job/sync/:arid", s.sync)
	s.engine.GET("/job/kill/:arid", s.sync)
	s.engine.GET("/jobs", s.getJobs)

	if err := s.engine.Run(port); err != nil {
		panic(err)
	}
}

func (s *Server) getTx(c *gin.Context) {
	arid := c.Param("arid")
	// TODO

	c.JSON(http.StatusOK, gin.H{
		"arid": arid,
	})
}

func (s *Server) getTxField(c *gin.Context) {
	arid := c.Param("arid")
	field := c.Param("field")
	// TODO

	c.JSON(http.StatusOK, gin.H{
		"arid":  arid,
		"field": field,
	})
}

func (s *Server) submitTx(c *gin.Context) {
	// TODO
	c.JSON(http.StatusOK, gin.H{})
}
