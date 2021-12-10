package seeding

import (
	"github.com/everFinance/goar/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) broadcast(c *gin.Context) {
	arid := c.Param("arid")

	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}

	if err := s.jobManager.RegisterJob(arid, jobTypeBroadcast, int64(len(s.peers))); err != nil {
		c.JSON(http.StatusBadGateway, err.Error())
		return
	}

	c.JSON(http.StatusOK, "ok")
}

func (s *Server) sync(c *gin.Context) {
	arid := c.Param("arid")
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}

	if err := s.jobManager.RegisterJob(arid, jobTypeSync, int64(len(s.peers))); err != nil {
		c.JSON(http.StatusBadGateway, err.Error())
		return
	}

	c.JSON(http.StatusOK, "ok")
}

func (s *Server) killJob(c *gin.Context) {
	arid := c.Param("arid")
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}
	s.jobManager.CloseJob(arid)
	c.JSON(http.StatusOK, "ok")
}

func (s *Server) getJob(c *gin.Context) {
	arid := c.Param("arid")
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		c.JSON(http.StatusBadRequest, "arId incorrect")
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": s.jobManager.GetJob(arid)})
}

func (s *Server) getJobs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"jobs": s.jobManager.GetJobs()})
}
