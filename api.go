package arseeding

import (
	"encoding/json"
	"fmt"
	"github.com/everFinance/arseeding/common"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/everpay-go/account"
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
	r.Use(common.CORSMiddleware())
	v1 := r.Group("/")
	{
		// Compatible arweave http api
		v1.POST("tx", s.submitTx)
		v1.POST("chunk", s.submitChunk)
		v1.GET("tx/:arid/offset", s.getTxOffset)
		v1.GET("/tx/:arid", s.getTx)
		v1.GET("chunk/:offset", s.getChunk)
		v1.GET("tx/:arid/:field", s.getTxField)
		v1.GET("/info", s.getInfo)
		v1.GET("/tx_anchor", s.getAnchor)
		v1.GET("/price/:size", s.getTxPrice)
		v1.GET("/peers", s.getPeers)
		// proxy
		v2 := r.Group("/")
		{
			v2.Use(proxyArweaveGateway)
			v2.GET("/tx/:arid/status")
			v2.GET("/price/:size/:target")
			v2.GET("/block/hash/:hash")
			v2.GET("/block/height/:height")
			v2.GET("/current_block")
			v2.GET("/wallet/:address/balance")
			v2.GET("/wallet/:address/last_tx")
			v2.POST("/arql")
			v2.POST("/graphql")
			v2.GET("/tx/pending")
			v2.GET("/unconfirmed_tx/:arId")
		}

		// broadcast && sync tasks
		v1.POST("/job/:taskType/:arid", s.postTask) // todo need delete when update pay-server
		v1.POST("/task/:taskType/:arid", s.postTask)
		v1.POST("/task/kill/:taskType/:arid", s.killTask)
		v1.GET("/task/:taskType/:arid", s.getTask)
		v1.GET("/task/cache", s.getCacheTasks)

		// ANS-104 bundle Data api
		v1.GET("/bundle/bundler", s.getBundler)
		v1.POST("/bundle/tx/:currency", s.submitItem)
		if s.NoFee {
			v1.POST("/bundle/tx", s.submitItem)
		}
		v1.GET("/bundle/tx/:itemId", s.getItemMeta) // get item meta, without data
		v1.GET("/bundle/itemIds/:arId", s.getItemIdsByArId)
		v1.GET("/bundle/fees", s.bundleFees)
		v1.GET("/bundle/fee/:size/:currency", s.bundleFee)
		v1.GET("/bundle/orders/:signer", s.getOrders)
		v1.GET("/:id", s.getDataByGW) // get arTx data or bundleItem data
	}

	if err := r.Run(port); err != nil {
		panic(err)
	}
}

func (s *Arseeding) submitTx(c *gin.Context) {
	arTx := types.Transaction{}
	if c.Request.Body == nil {
		errorResponse(c, "chunk data can not be null")
		return
	}
	by, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}
	defer c.Request.Body.Close()

	if err := json.Unmarshal(by, &arTx); err != nil {
		errorResponse(c, err.Error())
		return
	}
	// save tx to local
	if err = s.SaveSubmitTx(arTx); err != nil {
		errorResponse(c, err.Error())
		return
	}

	// register broadcast submit tx
	if err := s.registerTask(arTx.ID, schema.TaskTypeBroadcastMeta); err != nil {
		errorResponse(c, err.Error())
		return
	}
}

func (s *Arseeding) submitChunk(c *gin.Context) {
	chunk := types.GetChunk{}
	if c.Request.Body == nil {
		errorResponse(c, "chunk data can not be null")
		return
	}

	by, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}
	defer c.Request.Body.Close()

	if err := json.Unmarshal(by, &chunk); err != nil {
		errorResponse(c, err.Error())
		return
	}

	if err := s.SaveSubmitChunk(chunk); err != nil {
		errorResponse(c, err.Error())
		return
	}
}

func (s *Arseeding) getTxOffset(c *gin.Context) {
	arId := c.Param("arid")
	if len(arId) == 0 {
		errorResponse(c, "invalid_address")
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
		errorResponse(c, err.Error())
		return
	}

	chunk, err := s.store.LoadChunk(chunkOffset)
	if err != nil {
		if err == schema.ErrNotExist {
			c.Data(404, "text/html; charset=utf-8", []byte("Not Found"))
			return
		}
		errorResponse(c, err.Error())
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
	}

	// get from arweave gateway
	log.Debug("get from local failed, proxy to arweave gateway", "err", err, "arId", id)
	proxyArweaveGateway(c)
}

func (s *Arseeding) getTxField(c *gin.Context) {
	arid := c.Param("arid")
	field := c.Param("field")
	txMeta, err := s.store.LoadTxMeta(arid)
	if err != nil {
		log.Debug("get from local failed, proxy to arweave gateway", "err", err, "arId", arid, "field", field)
		proxyArweaveGateway(c)
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
		data, err := txDataByMeta(txMeta, s.store)
		if err != nil {
			c.JSON(400, err.Error())
			return
		}
		c.Data(200, "text/html; charset=utf-8", []byte(utils.Base64Encode(data)))

	case "data.json", "data.txt", "data.pdf":
		data, err := txDataByMeta(txMeta, s.store)
		if err != nil {
			errorResponse(c, err.Error())
			return
		}
		typ := strings.Split(field, ".")[1]
		c.Data(200, fmt.Sprintf("application/%s; charset=utf-8", typ), data)

	case "data.png", "data.jpeg", "data.gif":
		data, err := txDataByMeta(txMeta, s.store)
		if err != nil {
			errorResponse(c, err.Error())
			return
		}
		typ := strings.Split(field, ".")[1]
		c.Data(200, fmt.Sprintf("image/%s; charset=utf-8", typ), data)
	case "data.mp4":
		data, err := txDataByMeta(txMeta, s.store)
		if err != nil {
			errorResponse(c, err.Error())
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
		errorResponse(c, "invalid_field")
	}
}

func (s *Arseeding) getInfo(c *gin.Context) {
	info := s.cache.GetInfo()
	c.JSON(http.StatusOK, info)
}

func (s *Arseeding) getAnchor(c *gin.Context) {
	anchor := s.cache.GetAnchor()
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(anchor))
}

func (s *Arseeding) getTxPrice(c *gin.Context) {
	dataSize, err := strconv.ParseInt(c.Param("size"), 10, 64)
	if err != nil {
		errorResponse(c, err.Error())
	}
	fee := s.cache.GetFee()
	// totPrice = chunkNum*deltaPrice(fee for per chunk) + basePrice
	totPrice := calculatePrice(fee, dataSize)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(totPrice))
}

func (s *Arseeding) getPeers(c *gin.Context) {
	log.Debug("peers len", "len", len(s.cache.GetPeers()))
	c.JSON(http.StatusOK, s.cache.GetPeers())
}

func txDataByMeta(txMeta *types.Transaction, db *Store) ([]byte, error) {
	size, err := strconv.ParseUint(txMeta.DataSize, 10, 64)
	if err != nil {
		return nil, err
	}
	// When data is bigger than 12MiB return statusCode == 400, use chunk
	if size > 50*128*1024 {
		return nil, schema.ErrDataTooBig
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
	if err != nil {
		return nil, err
	}
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

func calculatePrice(fee schema.ArFee, dataSize int64) string {
	count := int64(0)
	if dataSize > 0 {
		count = (dataSize-1)/types.MAX_CHUNK_SIZE + 1
	}

	totPrice := fee.Base + count*fee.PerChunk
	return fmt.Sprintf("%d", totPrice)
}

// about task-manager

func (s *Arseeding) postTask(c *gin.Context) {
	arid := c.Param("arid")
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		errorResponse(c, "arId incorrect")
		return
	}
	tkType := c.Param("taskType")
	if !strings.Contains(schema.TaskTypeSync+schema.TaskTypeBroadcast+schema.TaskTypeBroadcastMeta, tkType) {
		errorResponse(c, "tktype not exist")
		return
	}

	if err = s.registerTask(arid, tkType); err != nil {
		errorResponse(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, "ok")
}

func (s *Arseeding) killTask(c *gin.Context) {
	arid := c.Param("arid")
	tktype := c.Param("taskType")
	if !strings.Contains(schema.TaskTypeSync+schema.TaskTypeBroadcast+schema.TaskTypeBroadcastMeta, tktype) {
		errorResponse(c, "tktype not exist")
		return
	}
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		errorResponse(c, "arId incorrect")
		return
	}
	err = s.taskMg.CloseTask(arid, tktype)
	if err != nil {
		c.JSON(http.StatusNotFound, err.Error())
	} else {
		c.JSON(http.StatusOK, "ok")
	}
}

func (s *Arseeding) getTask(c *gin.Context) {
	arid := c.Param("arid")
	tktype := c.Param("taskType")
	if !strings.Contains(schema.TaskTypeSync+schema.TaskTypeBroadcast+schema.TaskTypeBroadcastMeta, tktype) {
		errorResponse(c, "tktype not exist")
		return
	}
	txHash, err := utils.Base64Decode(arid)
	if err != nil || len(txHash) != 32 {
		errorResponse(c, "arId incorrect")
		return
	}
	// get from cache
	tk := s.taskMg.GetTask(arid, tktype)
	if tk != nil {
		c.JSON(http.StatusOK, tk)
		return
	}

	// get from db
	tk, err = s.store.LoadTask(assembleTaskId(arid, tktype))
	if err != nil {
		errorResponse(c, err.Error())
	} else {
		c.JSON(http.StatusOK, tk)
	}
}

func (s *Arseeding) getCacheTasks(c *gin.Context) {
	tks := s.taskMg.GetTasks()
	total := len(tks)
	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"tasks": tks,
	})
}

func (s *Arseeding) registerTask(arId, tktype string) error {
	s.taskMg.AddTask(arId, tktype)
	if err := s.store.PutTaskPendingPool(assembleTaskId(arId, tktype)); err != nil {
		s.taskMg.DelTask(arId, tktype)
		log.Error("PutTaskPendingPool(tktype, arTx.ID)", "err", err, "arId", arId, "tktype", tktype)
		return err
	}

	s.taskMg.PutToTkChan(arId, tktype)
	return nil
}

func (s *Arseeding) getBundler(c *gin.Context) {
	c.JSON(http.StatusOK, schema.ResBundler{Bundler: s.bundler.Signer.Address})
}

func (s *Arseeding) submitItem(c *gin.Context) {
	if c.GetHeader("Content-Type") != "application/octet-stream" {
		errorResponse(c, "Wrong body type")
		return
	}
	if c.Request.Body == nil {
		errorResponse(c, "can not submit null bundle item")
		return
	}

	defer c.Request.Body.Close()

	itemBinary := make([]byte, 0, 256*1024)
	buf := make([]byte, 256*1024) // todo add to temp file
	for {
		if len(itemBinary) > schema.AllowMaxItemSize {
			err := fmt.Errorf("allow max item size is 100 MB")
			errorResponse(c, err.Error())
			return
		}

		n, err := c.Request.Body.Read(buf)
		if err != nil && err != io.EOF {
			errorResponse(c, "read req failed")
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
		errorResponse(c, "decode item binary failed")
		return
	}
	currency := c.Param("currency")

	// process bundleItem
	ord, err := s.ProcessSubmitItem(*item, currency)
	if err != nil {
		errorResponse(c, err.Error())
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

func (s *Arseeding) getItemMeta(c *gin.Context) {
	id := c.Param("itemId")
	// could be bundle item id
	meta, err := s.store.LoadItemMeta(id)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, meta)
}

func (s *Arseeding) getItemIdsByArId(c *gin.Context) {
	arId := c.Param("arId")
	itemIds, err := s.store.LoadArIdToItemIds(arId)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, itemIds)
}

func (s *Arseeding) bundleFee(c *gin.Context) {
	size := c.Param("size")
	symbol := c.Param("currency")
	numSize, err := strconv.Atoi(size)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	respFee, err := s.CalcItemFee(symbol, int64(numSize))
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, respFee)
}

func (s *Arseeding) getOrders(c *gin.Context) {
	signer := c.Param("signer")
	_, signerAddr, err := account.IDCheck(signer)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}

	cursorId, err := strconv.ParseInt(c.DefaultQuery("cursorId", "0"), 10, 64)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}
	num := 200
	orders, err := s.wdb.GetOrdersBySigner(signerAddr, cursorId, num)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (s *Arseeding) bundleFees(c *gin.Context) {
	c.JSON(http.StatusOK, s.bundlePerFeeMap)
}

func (s *Arseeding) getDataByGW(c *gin.Context) {
	id := c.Param("id")
	txMeta, err := s.store.LoadTxMeta(id)
	if err == nil { // find id is arId
		data, err := txDataByMeta(txMeta, s.store)
		if err != nil {
			internalErrorResponse(c, err.Error())
			return
		}
		c.Data(200, fmt.Sprintf("%s; charset=utf-8", getTagValue(txMeta.Tags, "Content-Type")), data)
		return
	}

	// not arId
	itemBinary, err := s.store.LoadItemBinary(id)
	if err == nil { // id is bundle item id
		item, err := utils.DecodeBundleItem(itemBinary)
		if err != nil {
			internalErrorResponse(c, err.Error())
			return
		}
		data, err := utils.Base64Decode(item.Data)
		if err != nil {
			internalErrorResponse(c, err.Error())
			return
		}
		c.Data(200, fmt.Sprintf("%s; charset=utf-8", getTagValue(item.Tags, "Content-Type")), data)
		return
	}

	// get from arweave gateway
	proxyArweaveGateway(c)
}

func getTagValue(tags []types.Tag, name string) string {
	for _, tg := range tags {
		if tg.Name == name {
			return tg.Value
		}
	}
	return ""
}

func errorResponse(c *gin.Context, err string) {
	// client error
	c.JSON(http.StatusBadRequest, schema.RespErr{
		Err: err,
	})
}

func internalErrorResponse(c *gin.Context, err string) {
	// client error
	c.JSON(http.StatusInternalServerError, schema.RespErr{
		Err: err,
	})
}
