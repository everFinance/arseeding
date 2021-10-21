package seeding

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) broadcast(c *gin.Context) {
	arid := c.Param("arid")
	// TODO
	c.JSON(http.StatusOK, gin.H{"broadcast": arid})
}

func (s *Server) sync(c *gin.Context) {
	arid := c.Param("arid")
	// TODO
	c.JSON(http.StatusOK, gin.H{"sync": arid})
}

func (s *Server) getJobs(c *gin.Context) {
	// TODO
	c.JSON(http.StatusOK, gin.H{"jobs": ""})
}
