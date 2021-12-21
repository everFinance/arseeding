package arseeding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) runAPI(port string) {
	r := s.engine
	v1 := r.Group("/")
	{
		// Compatible arweave http api
		v1.POST("tx", s.submitTx)
		v1.POST("chunk", s.submitChunk)
		v1.GET("tx/:arid/offset", s.getTxOffset)
		v1.GET("/tx/:arid", s.getTx)
		v1.GET("chunk/:offset", s.getChunk)
		v1.GET("tx/:arid/:field", s.getTxField)
		// proxy
		v2 := r.Group("/")
		{
			v2.Use(proxyArweaveGateway)

			v2.GET("/info")
			v2.GET("/tx/:arid/status")
			v2.GET("/:arid")
			v2.GET("/price/:size")
			v2.GET("/price/:size/:target")
			v2.GET("/block/hash/:hash")
			v2.GET("/block/height/:height")
			v2.GET("/current_block")
			v2.GET("/wallet/:address/balance")
			v2.GET("/wallet/:address/last_tx")
			v2.GET("/peers")
			v2.GET("/tx_anchor")
			v2.POST("/arql")
			v2.POST("/graphql")
			v2.GET("/tx/pending")
			v2.GET("/unconfirmed_tx/:arId")
		}

		// broadcast && sync jobs
		v1.POST("/job/broadcast/:arid", s.broadcast)
		v1.POST("/job/sync/:arid", s.sync)
		v1.POST("/job/kill/:arid/:jobType", s.killJob)
		v1.GET("/job/:arid/:jobType", s.getJob)
		v1.GET("/cache/jobs", s.getCacheJobs)
	}

	if err := r.Run(port); err != nil {
		panic(err)
	}
}

func (s *Server) submitTx(c *gin.Context) {
	arTx := types.Transaction{}
	if c.Request.Body == nil {
		c.JSON(http.StatusBadRequest, "chunk data can not be null")
		return
	}
	by, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	defer c.Request.Body.Close()

	if err := json.Unmarshal(by, &arTx); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	if err := s.processSubmitTx(arTx); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	// proxy to arweave
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(by)))
	proxyArweaveGateway(c)
}

func (s *Server) submitChunk(c *gin.Context) {
	chunk := types.GetChunk{}
	if c.Request.Body == nil {
		c.JSON(http.StatusBadRequest, "chunk data can not be null")
		return
	}

	by, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	defer c.Request.Body.Close()

	if err := json.Unmarshal(by, &chunk); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	if err := s.processSubmitChunk(chunk); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	// proxy to arweave
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(by)))
	proxyArweaveGateway(c)
}

func (s *Server) getTxOffset(c *gin.Context) {
	arId := c.Param("arid")
	if len(arId) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_address"})
		return
	}
	txMeta, err := s.store.LoadTxMeta(arId)
	if err != nil {
		c.Data(404, "text/html; charset=utf-8", []byte("Not Found"))
		return
	}
	offset, err := s.store.LoadTxDataEndOffSet(txMeta.DataRoot, txMeta.DataSize)
	if err != nil {
		c.Data(404, "text/html; charset=utf-8", []byte("Not Found"))
		return
	}

	txOffset := &types.TransactionOffset{
		Size:   txMeta.DataSize,
		Offset: fmt.Sprintf("%d", offset),
	}
	c.JSON(http.StatusOK, txOffset)
}

func (s *Server) getChunk(c *gin.Context) {
	offset := c.Param("offset")
	chunkOffset, err := strconv.ParseUint(offset, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	chunk, err := s.store.LoadChunk(chunkOffset)
	if err != nil {
		if err == ErrNotExist {
			c.Data(404, "text/html; charset=utf-8", []byte("Not Found"))
			return
		}
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, chunk)
}

func (s *Server) getTx(c *gin.Context) {
	arid := c.Param("arid")
	arTx, err := s.store.LoadTxMeta(arid)
	if err != nil {
		if err == ErrNotExist {
			c.Data(404, "text/html; charset=utf-8", []byte("Not Found"))
			return
		}
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, arTx)
}

func (s *Server) getTxField(c *gin.Context) {
	arid := c.Param("arid")
	field := c.Param("field")
	txMeta, err := s.store.LoadTxMeta(arid)
	if err != nil {
		if err == ErrNotExist {
			c.JSON(404, "not found")
			return
		}
		c.JSON(404, err.Error()) // not found
		return
	}

	switch field {
	case "id":
		c.JSON(http.StatusOK, txMeta.ID)
	case "last_tx":
		c.JSON(http.StatusOK, txMeta.LastTx)
	case "owner":
		c.JSON(http.StatusOK, txMeta.Owner)
	case "tags":
		c.JSON(http.StatusOK, txMeta.Tags)
	case "target":
		c.JSON(http.StatusOK, txMeta.Target)
	case "quantity":
		c.JSON(http.StatusOK, txMeta.Quantity)
	case "data":
		data, err := txMetaData(txMeta, s.store)
		if err != nil {
			c.JSON(400, err.Error())
			return
		}
		c.Data(200, "text/html; charset=utf-8", []byte(utils.Base64Encode(data)))

	case "data.json", "data.txt", "data.pdf":
		data, err := txMetaData(txMeta, s.store)
		if err != nil {
			c.JSON(400, err.Error())
			return
		}
		// c.Data(200,"text/html; charset=utf-8",data)
		typ := strings.Split(field, ".")[1]
		c.Data(200, fmt.Sprintf("application/%s; charset=utf-8", typ), data)

	case "data.png", "data.jpeg", "data.gif":
		data, err := txMetaData(txMeta, s.store)
		if err != nil {
			c.JSON(400, err.Error())
			return
		}
		typ := strings.Split(field, ".")[1]
		c.Data(200, fmt.Sprintf("image/%s; charset=utf-8", typ), data)
	case "data.mp4":
		data, err := txMetaData(txMeta, s.store)
		if err != nil {
			c.JSON(400, err.Error())
			return
		}
		c.Data(200, "video/mpeg4; charset=utf-8", data)
	case "data_root":
		c.JSON(http.StatusOK, txMeta.DataRoot)
	case "data_size":
		c.JSON(http.StatusOK, txMeta.DataSize)
	case "reward":
		c.JSON(http.StatusOK, txMeta.Reward)
	case "signature":
		c.JSON(http.StatusOK, txMeta.Signature)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_field"})
	}
}

func txMetaData(txMeta *types.Transaction, db *Store) ([]byte, error) {
	size, err := strconv.ParseUint(txMeta.DataSize, 10, 64)
	if err != nil {
		return nil, err
	}
	// When data is bigger than 12MiB return statusCode == 400, use chunk
	if size > 50*128*1024 {
		return nil, errors.New("tx_data_too_big")
	}

	data, err := getData(txMeta.DataRoot, txMeta.DataSize, db)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func getData(dataRoot, dataSize string, db *Store) ([]byte, error) {
	size, err := strconv.ParseUint(dataSize, 10, 64)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 0, size)
	txDataEndOffset, err := db.LoadTxDataEndOffSet(dataRoot, dataSize)
	startOffset := txDataEndOffset - size + 1
	for i := 0; uint64(i)+startOffset < txDataEndOffset; {
		chunkStartOffset := startOffset + uint64(i)
		chunk, err := db.LoadChunk(chunkStartOffset)
		if err != nil {
			return nil, err
		}
		chunkData, err := utils.Base64Decode(chunk.Chunk)
		if err != nil {
			return nil, err
		}
		data = append(data, chunkData...)
		i += len(chunkData)
	}
	return data, nil
}

func proxyArweaveGateway(c *gin.Context) {
	var proxyUrl = new(url.URL)
	proxyUrl.Scheme = "https"
	proxyUrl.Host = "arweave.net"

	proxy := httputil.NewSingleHostReverseProxy(proxyUrl)
	proxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
