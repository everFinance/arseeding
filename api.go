package arseeding

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
)

func (s *Arseeding) runAPI(port string) {
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

			// v2.GET("/info")
			v2.GET("/tx/:arid/status")
			// v2.GET("/:arid")
			// v2.GET("/price/:size")
			v2.GET("/price/:size/:target")
			v2.GET("/block/hash/:hash")
			v2.GET("/block/height/:height")
			v2.GET("/current_block")
			v2.GET("/wallet/:address/balance")
			v2.GET("/wallet/:address/last_tx")
			v2.GET("/peers")
			// v2.GET("/tx_anchor")
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
		v1.GET("/info", s.getInfo)
		v1.GET("/tx_anchor", s.getAnchor)
		v1.GET("/price/:size", s.getTxPrice)

		// ANS-104 bundle Data api
		v1.POST("/bundle/tx/:currency", s.submitItem)
		v1.GET("/bundle/tx/:id", s.getItemMeta) // get item meta, without data
		v1.GET("/bundle/fees", s.bundleFees)
		v1.GET("/bundle/fee/:size/:symbol", s.bundleFee)
		v1.GET("/:id", s.getData) // get arTx data or bundleItem data
	}

	if err := r.Run(port); err != nil {
		panic(err)
	}
}

func (s *Arseeding) submitTx(c *gin.Context) {
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

	if err := s.broadcastSubmitTx(arTx); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
}

func (s *Arseeding) submitChunk(c *gin.Context) {
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
}

func (s *Arseeding) getTxOffset(c *gin.Context) {
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

func (s *Arseeding) getChunk(c *gin.Context) {
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

func (s *Arseeding) getTx(c *gin.Context) {
	id := c.Param("arid")
	arTx, err := s.store.LoadTxMeta(id)
	if err == nil {
		c.JSON(http.StatusOK, arTx)
		return
	} else if err != ErrNotExist {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	// get from arweave gateway
	proxyArweaveGateway(c)
}

func (s *Arseeding) getTxField(c *gin.Context) {
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
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.ID))
	case "last_tx":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.LastTx))
	case "owner":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Owner))
	case "tags":
		c.JSON(http.StatusOK, txMeta.Tags)
	case "target":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Target))
	case "quantity":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Quantity))
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
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.DataRoot))
	case "data_size":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.DataSize))
	case "reward":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Reward))
	case "signature":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Signature))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"status": 400, "error": "invalid_field"})
	}
}

func (s *Server) getInfo(c *gin.Context) {
	info, err := s.cache.GetInfo()
	if err != nil {
		if err == ErrNotExist {
			c.Data(404, "text/html; charset=utf-8", []byte("Not Found"))
			return
		}
		c.JSON(http.StatusBadRequest, err.Error())
	}
	c.JSON(http.StatusOK, info)
}

func (s *Server) getAnchor(c *gin.Context) {
	anchor, err := s.cache.GetAnchor()
	if err != nil {
		if err == ErrNotExist {
			c.Data(404, "text/html; charset=utf-8", []byte("Not Found"))
			return
		}
		c.JSON(http.StatusBadRequest, err.Error())
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(anchor))
}

func (s *Server) getTxPrice(c *gin.Context) {
	dataSize, err := strconv.ParseInt(c.Param("size"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
	price, err := s.cache.GetPrice()
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
	// totPrice = chunkNum*deltaPrice(price for per chunk) + basePrice
	totPrice := calculatePrice(*price, dataSize)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(totPrice))
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
	if size == 0 {
		return []byte{}, nil
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
	directer := func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = "arweave.net"
		req.Host = "arweave.net"
	}
	proxy := &httputil.ReverseProxy{Director: directer}

	proxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

func calculatePrice(price TxPrice, dataSize int64) string {

	var chunkSize int64 = 256 * 1024
	totPrice := price.basePrice
	chunkNum := dataSize / chunkSize
	if dataSize%chunkSize != 0 {
		chunkNum += 1
	}
	totPrice += chunkNum * price.perChunkPrice
	return fmt.Sprintf("%d", totPrice)
}

func (s *Arseeding) submitItem(c *gin.Context) {
	if c.GetHeader("content-type") != "application/octet-stream" {
		c.JSON(http.StatusBadRequest, "Wrong body type")
		return
	}
	if c.Request.Body == nil {
		c.JSON(http.StatusBadRequest, "can not submit null bundle item")
		return
	}

	defer c.Request.Body.Close()

	itemBinary := make([]byte, 0, 256*1024)
	buf := make([]byte, 20*1024) // todo add to temp file
	for {
		if len(itemBinary) > schema.AllowMaxItemSize {
			err := fmt.Errorf("allow max item size is 100 MB")
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}

		n, err := c.Request.Body.Read(buf)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusBadRequest, "read req failed")
			log.Error("read req failed", "err", err)
			return
		}

		if n == 0 {
			break
		}
		itemBinary = append(itemBinary, buf[:n]...)
	}

	// decode
	item, err := utils.DecodeBundleItem(itemBinary)
	if err != nil {
		c.JSON(http.StatusBadRequest, "decode item binary failed")
		return
	}
	currency := c.Param("currency")

	// process bundleItem
	ord, err := s.processSubmitBundleItem(*item, currency)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, schema.RespOrder{
		ItemId:             ord.ItemId,
		Bundler:            s.bundler.Signer.Address,
		Currency:           ord.Currency,
		Decimals:           ord.Decimals,
		Fee:                ord.Fee,
		PaymentExpiredTime: ord.PaymentExpiredTime,
		ExpectedBlock:      ord.ExpectedBlock,
	})
}

func (s *Arseeding) bundleFee(c *gin.Context) {
	size := c.Param("size")
	symbol := c.Param("symbol")
	numSize, err := strconv.Atoi(size)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	respFee, err := s.calcItemFee(symbol, int64(numSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, respFee)
}

func (s *Arseeding) bundleFees(c *gin.Context) {
	c.JSON(http.StatusOK, s.bundlePerFeeMap)
}

func (s *Arseeding) getData(c *gin.Context) {
	id := c.Param("id")
	txMeta, err := s.store.LoadTxMeta(id)
	if err == nil { // find id is arId
		data, err := txMetaData(txMeta, s.store)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.Data(200, fmt.Sprintf("%s; charset=utf-8", getTagValue(txMeta.Tags, "Content-Type")), data)
		return
	} else if err != ErrNotExist {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// not arId
	itemBinary, err := s.store.LoadItemBinary(id)
	if err == nil { // id is bundle item id
		item, err := utils.DecodeBundleItem(itemBinary)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		data, err := utils.Base64Decode(item.Data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.Data(200, fmt.Sprintf("%s; charset=utf-8", getTagValue(item.Tags, "Content-Type")), data)
		return
	} else if err != ErrNotExist {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	// get from arweave gateway
	proxyArweaveGateway(c)
}

func (s *Arseeding) getItemMeta(c *gin.Context) {
	id := c.Param("id")
	// could be bundle item id
	meta, err := s.store.LoadItemMeta(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(200, meta)
}

func getTagValue(tags []types.Tag, name string) string {
	for _, tg := range tags {
		if tg.Name == name {
			return tg.Value
		}
	}
	return ""
}
