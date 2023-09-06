package arseeding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/go-everpay/account"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/everFinance/goether"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/handlers"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"io"
	"io/ioutil"
	gLog "log"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	tmpFileMap     = make(map[string]int64) // key->v : fileName -> fileUsedCnt
	tmpFileMapLock sync.Mutex
)

func (s *Arseeding) runAPI(port string) {
	r := s.engine
	r.Use(CORSMiddleware())
	if s.EnableManifest {
		r.Use(ManifestMiddleware(s))
	}

	if !s.NoFee {
		r.Use(LimiterMiddleware(300000, "M", s.config.GetIPWhiteList()))
	}
	v1 := r.Group("/")
	{
		v1.Any("/", s.arseedInfo)
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

		v1.GET("/bundle/tx/:itemId", s.getItemMeta) // get item meta, without data
		v1.GET("/bundle/tx/:itemId/:field", s.getItemField)
		v1.GET("/bundle/itemIds/:arId", s.getItemIdsByArId)
		v1.GET("/bundle/fees", s.bundleFees)
		v1.GET("/bundle/fee/:size/:currency", s.bundleFee)
		v1.GET("/bundle/orders/:signer", s.getOrders)
		v1.GET("/:id", s.dataRoute)  // get arTx data or bundleItem data
		v1.HEAD("/:id", s.dataRoute) // get arTx data or bundleItem data

		if s.EnableManifest {
			v1.POST("/manifest_url/:id", s.setManifestUrl)
		}

		// submit native data with X-API-KEY
		v1.POST("/bundle/data/:currency", s.submitNativeData)
		v1.GET("/bundle/orders", s.getOrdersByApiKey) // http header need X-API-KEY

		// apikey
		v1.GET("/apikey_info/:address", s.getApiKeyInfo)
		v1.GET("/apikey/:timestamp/:signature", s.getApiKey)
		v1.GET("/apikey_records/deposit/:address", s.getApikeyDepositRecords)

		// statistic
		v1.GET("/statistic/realtime", s.getRealTimeOrderStatistic)
		v1.GET("/statistic/range", s.getOrderStatisticByDate)
	}

	go func() {
		gLog.Fatal(http.ListenAndServe(":8081", handlers.CompressHandler(http.DefaultServeMux)))
	}()
	gLog.Printf("you can now open http://localhost:8080/debug/charts/ in your browser")

	if err := r.Run(port); err != nil {
		panic(err)
	}
}

func (s *Arseeding) arseedInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Name":          "Arseeding",
		"Version":       "v1.2.0",
		"Documentation": "https://web3infra.dev",
		"ConcurrentNum": s.config.Param.ChunkConcurrentNum,
	})
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
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf("%d", totPrice)))
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
	// When data is bigger than 50MiB return statusCode == 400, use chunk
	if size > schema.AllowMaxRespDataSize {
		return nil, schema.ErrDataTooBig
	}

	data, err := getArTxData(txMeta.DataRoot, txMeta.DataSize, db)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// todo need stream
func getArTxData(dataRoot, dataSize string, db *Store) ([]byte, error) {
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
	c.Writer.Header().Del("Access-Control-Allow-Origin")
	directer := func(req *http.Request) {
		req.URL.Scheme = "https"
		req.URL.Host = "arweave.net"
		req.Host = "arweave.net"
	}
	proxy := &httputil.ReverseProxy{Director: directer}

	proxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

func calculatePrice(fee schema.ArFee, dataSize int64) int64 {
	count := int64(0)
	if dataSize > 0 {
		count = (dataSize-1)/types.MAX_CHUNK_SIZE + 1
	}

	totPrice := fee.Base + count*fee.PerChunk
	return totPrice
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

func (s *Arseeding) processApikeySpendBal(currency, apikey string, dataSize int64) error {
	apikeyDetail, err := s.wdb.GetApiKeyDetail(apikey)
	if err != nil {
		return err
	}

	// exist currency balance
	curBal, ok := apikeyDetail.TokenBalance[strings.ToUpper(currency)]
	if !ok {
		return errors.New("apiKey not the currency balance")
	}
	curBalDe, err := decimal.NewFromString(curBal.(string))
	if err != nil {
		return err
	}
	// calc fee
	fee, err := s.CalcItemFee(currency, dataSize)
	if err != nil {
		return err
	}
	feeDe, err := decimal.NewFromString(fee.FinalFee)
	if err != nil {
		return err
	}
	endBalDe := curBalDe.Sub(feeDe)
	if endBalDe.LessThan(decimal.NewFromInt(0)) {
		// balance is insufficient
		return errors.New("balance is insufficient")
	}

	// update db tokenBalance
	apikeyDetail.TokenBalance[strings.ToUpper(currency)] = endBalDe.String()
	if err = s.wdb.UpdateApikeyTokenBal(apikeyDetail.Address, apikeyDetail.TokenBalance); err != nil {
		log.Error("UpdateApikeyTokenBal", "err", err)
		return err
	}
	return nil
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
	itemBinaryFile, err := os.CreateTemp(schema.TmpFileDir, "arseedsubmit-")
	if err != nil {
		c.Request.Body.Close()
		errorResponse(c, err.Error())
		return
	}
	defer func() {
		c.Request.Body.Close()
		itemBinaryFile.Close()
		os.Remove(itemBinaryFile.Name())
	}()

	var itemBuf bytes.Buffer
	var item *types.BundleItem
	// write up to schema.AllowStreamMinItemSize to memory
	size, err := setItemData(c, itemBinaryFile, &itemBuf)
	if err != nil && err != io.EOF {
		errorResponse(c, err.Error())
		return
	}

	if size > schema.SubmitMaxSize {
		errorResponse(c, schema.ErrDataTooBig.Error())
		return
	}
	if size > schema.AllowStreamMinItemSize { // the body size > schema.AllowStreamMinItemSize, need write to tmp file
		item, err = utils.DecodeBundleItemStream(itemBinaryFile)
	} else {
		item, err = utils.DecodeBundleItem(itemBuf.Bytes())
	}
	defer func() {
		if item.DataReader != nil {
			item.DataReader.Close()
			os.Remove(item.DataReader.Name())
		}
	}()

	if err != nil {
		errorResponse(c, "decode item binary failed")
		return
	}

	currency := c.Param("currency")
	// check whether noFee mode
	noFee := false
	// if has apikey
	apikey := c.GetHeader("X-API-KEY")
	hasApikey := false
	if len(apikey) > 0 {
		if err := s.processApikeySpendBal(currency, apikey, size); err != nil {
			errorResponse(c, err.Error())
			return
		}
		// currency has balance
		hasApikey = true
	}

	if s.NoFee || hasApikey {
		noFee = true
	}

	// process bundleItem
	needSort := isSortItems(c)
	ord, err := s.ProcessSubmitItem(*item, currency, noFee, apikey, needSort, size)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, schema.RespOrder{
		ItemId:             ord.ItemId,
		Size:               ord.Size,
		Bundler:            s.bundler.Signer.Address,
		Currency:           ord.Currency,
		Decimals:           ord.Decimals,
		Fee:                ord.Fee,
		PaymentExpiredTime: ord.PaymentExpiredTime,
		ExpectedBlock:      ord.ExpectedBlock,
	})
}

func (s *Arseeding) submitNativeData(c *gin.Context) {
	apiKey := c.GetHeader("X-API-KEY")
	if len(apiKey) == 0 {
		errorResponse(c, "Wrong X-API-KEY")
		return
	}
	_, err := s.wdb.GetApiKeyDetail(apiKey)
	if err != nil {
		errorResponse(c, fmt.Sprintf("Wrong X-API-KEY: %s", err.Error()))
		return
	}

	// get all query and assemble tags
	queryMap := c.Request.URL.Query()
	// query key must include "Content-Type"
	if _, ok := queryMap["Content-Type"]; !ok {
		errorResponse(c, "Query params must include Content-Type")
		return
	}
	needSort := isSortItems(c)
	tags := make([]types.Tag, 0, len(queryMap))
	for k, values := range queryMap {
		for _, val := range values {
			tags = append(tags, types.Tag{
				Name:  k,
				Value: val,
			})
		}
	}

	if c.Request.Body == nil {
		errorResponse(c, "can not submit null native data")
		return
	}

	dataFile, err := os.CreateTemp(schema.TmpFileDir, "arseed-")
	if err != nil {
		c.Request.Body.Close()
		errorResponse(c, err.Error())
		return
	}
	defer func() {
		c.Request.Body.Close()
		dataFile.Close()
		os.Remove(dataFile.Name())
	}()
	var dataBuf bytes.Buffer
	var item types.BundleItem
	// write up to schema.AllowMaxNativeDataSize to memory
	size, err := setItemData(c, dataFile, &dataBuf)
	if err != nil && err != io.EOF {
		errorResponse(c, err.Error())
		return
	}
	if size > schema.SubmitMaxSize {
		errorResponse(c, schema.ErrDataTooBig.Error())
		return
	}

	if size > schema.AllowStreamMinItemSize { // the body size > schema.AllowStreamMinItemSize, need write to tmp file
		item, err = s.bundlerItemSigner.CreateAndSignItemStream(dataFile, "", "", tags)
	} else {
		item, err = s.bundlerItemSigner.CreateAndSignItem(dataBuf.Bytes(), "", "", tags)
	}

	if err != nil {
		errorResponse(c, "assemble bundle item failed")
		log.Error("s.bundlerItemSigner.CreateAndSignItem", "err", err)
		return
	}
	currency := c.Param("currency")
	// cal apikey balance
	if err := s.processApikeySpendBal(currency, apiKey, size); err != nil {
		errorResponse(c, err.Error())
		return
	}

	// process submit item
	order, err := s.ProcessSubmitItem(item, currency, true, apiKey, needSort, size)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, schema.RespItemId{ItemId: order.ItemId, Size: order.Size})
}

func (s *Arseeding) getOrdersByApiKey(c *gin.Context) {
	apiKey := c.GetHeader("X-API-KEY")
	_, err := s.wdb.GetApiKeyDetail(apiKey)
	if err != nil {
		errorResponse(c, "Wrong X-API-KEY")
		return
	}

	cursorId, err := strconv.ParseInt(c.DefaultQuery("cursorId", "0"), 10, 64)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("size", "0"))
	if err != nil {
		errorResponse(c, err.Error())
		return
	}
	MaxSize := 1000
	if pageSize <= 0 || pageSize > MaxSize {
		pageSize = MaxSize
	}

	sort := c.DefaultQuery("sort", "DESC")
	orders, err := s.wdb.GetOrdersByApiKey(apiKey, cursorId, pageSize, sort)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	results := make([]schema.RespGetOrder, 0, len(orders))
	for _, od := range orders {
		results = append(results, schema.RespGetOrder{
			ID: od.ID,
			RespOrder: schema.RespOrder{
				ItemId:             od.ItemId,
				Size:               od.Size,
				Bundler:            s.bundler.Signer.Address,
				Currency:           od.Currency,
				Decimals:           od.Decimals,
				Fee:                od.Fee,
				PaymentExpiredTime: od.PaymentExpiredTime,
				ExpectedBlock:      od.ExpectedBlock,
			},
			PaymentStatus: od.PaymentStatus,
			PaymentId:     od.PaymentId,
			OnChainStatus: od.OnChainStatus,
			Sort:          od.Sort,
		})
	}
	c.JSON(http.StatusOK, results)
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

func (s *Arseeding) getItemField(c *gin.Context) {
	id := c.Param("itemId")
	field := c.Param("field")
	txMeta, err := s.store.LoadItemMeta(id)
	if err != nil {
		notFoundResponse(c, err.Error())
		return
	}
	switch field {
	case "id":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Id))
	case "anchor":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Anchor))
	case "owner":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Owner))
	case "tags":
		c.JSON(http.StatusOK, txMeta.Tags)
	case "target":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Target))
	case "signature":
		c.Data(200, "text/html; charset=utf-8", []byte(txMeta.Signature))
	case "signatureType":
		c.Data(200, "text/html; charset=utf-8", []byte(strconv.Itoa(txMeta.SignatureType)))
	case "data", "data.json", "data.txt", "data.pdf", "data.png", "data.jpeg", "data.gif", "data.mp4":
		tags, dataReader, data, err := getBundleItemData(id, s.store)
		if err != nil {
			internalErrorResponse(c, err.Error())
			return
		}

		dataResponse(c, dataReader, data, tags, id)
	default:
		errorResponse(c, "invalid_field")
	}
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
		errorResponse(c, err.Error())
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
	num, err := strconv.ParseInt(c.DefaultQuery("num", "20"), 10, 64)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}

	orders, err := s.wdb.GetOrdersBySigner(signerAddr, cursorId, int(num))
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	results := make([]schema.RespGetOrder, 0, len(orders))
	for _, od := range orders {
		results = append(results, schema.RespGetOrder{
			ID: od.ID,
			RespOrder: schema.RespOrder{
				ItemId:             od.ItemId,
				Size:               od.Size,
				Bundler:            s.bundler.Signer.Address,
				Currency:           od.Currency,
				Decimals:           od.Decimals,
				Fee:                od.Fee,
				PaymentExpiredTime: od.PaymentExpiredTime,
				ExpectedBlock:      od.ExpectedBlock,
			},
			PaymentStatus: od.PaymentStatus,
			PaymentId:     od.PaymentId,
			OnChainStatus: od.OnChainStatus,
			Sort:          od.Sort,
		})
	}
	c.JSON(http.StatusOK, results)
}

func (s *Arseeding) bundleFees(c *gin.Context) {
	c.JSON(http.StatusOK, s.bundlePerFeeMap)
}

func (s *Arseeding) dataRoute(c *gin.Context) {
	txId := c.Param("id")
	tmpFileName := genTmpFileName(c.ClientIP(), txId)
	if existTmpFile(tmpFileName) {
		dataReader, err := os.Open(tmpFileName)
		if err != nil {
			internalErrorResponse(c, err.Error())
			return
		}
		item, err := s.store.LoadItemMeta(txId)
		if err != nil {
			internalErrorResponse(c, err.Error())
			return
		}
		dataResponse(c, dataReader, nil, item.Tags, txId)
		return
	}
	tags, dataReader, data, err := getArTxOrItemData(txId, s.store)
	switch err {
	case nil:
		// process manifest
		if s.EnableManifest && getTagValue(tags, schema.ContentType) == schema.ManifestType {
			mfUrl := expectedTxSandbox(txId)
			if _, err = s.wdb.GetManifestId(mfUrl); err == gorm.ErrRecordNotFound {
				// insert new record
				if err = s.wdb.InsertManifest(schema.Manifest{
					ManifestUrl: mfUrl,
					ManifestId:  txId,
				}); err != nil {
					internalErrorResponse(c, err.Error())
					return
				}
			}

			protocol := "https"
			if c.Request.TLS == nil {
				protocol = "http"
			}
			redirectUrl := fmt.Sprintf("%s://%s.%s", protocol, mfUrl, c.Request.Host)

			c.Redirect(302, redirectUrl)
		} else {
			dataResponse(c, dataReader, data, tags, txId)
		}

	case schema.ErrLocalNotExist:
		proxyArweaveGateway(c)
	default:
		internalErrorResponse(c, err.Error())
	}
}

func (s *Arseeding) setManifestUrl(c *gin.Context) {
	txId := c.Param("id")
	mfUrl := expectedTxSandbox(txId)
	if mfId, err := s.wdb.GetManifestId(mfUrl); err == nil {
		c.JSON(http.StatusOK, schema.Manifest{
			ManifestUrl: mfUrl,
			ManifestId:  mfId,
		})
		return
	}

	tags, err := getArTxOrItemTags(txId, s.store)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	if s.EnableManifest && getTagValue(tags, schema.ContentType) == schema.ManifestType {
		// insert new record
		if err = s.wdb.InsertManifest(schema.Manifest{
			ManifestUrl: mfUrl,
			ManifestId:  txId,
		}); err != nil {
			internalErrorResponse(c, err.Error())
			return
		}
	}

	c.JSON(http.StatusOK, schema.Manifest{
		ManifestUrl: mfUrl,
		ManifestId:  txId,
	})
}

func setItemData(c *gin.Context, tmpFile *os.File, itemBuf *bytes.Buffer) (size int64, err error) {
	size, err = io.CopyN(itemBuf, c.Request.Body, schema.AllowStreamMinItemSize+1)
	if err != nil && err != io.EOF {
		return
	}
	if size > schema.AllowStreamMinItemSize { // the body size > schema.AllowStreamMinItemSize, need write to tmp file

		size, err = io.Copy(tmpFile, io.MultiReader(itemBuf, c.Request.Body))
		if err != nil {
			return
		}
		// reset io stream to origin of the file
		_, err = tmpFile.Seek(0, 0)
		if err != nil {
			return
		}
	}
	return
}

func getTagValue(tags []types.Tag, name string) string {
	for _, tg := range tags {
		if tg.Name == name {
			return tg.Value
		}
	}
	return ""
}

func isSortItems(c *gin.Context) bool {
	if c.GetHeader("Sort") == "true" || c.GetHeader("sort") == "true" {
		return true
	}
	return false
}

func errorResponse(c *gin.Context, err string) {
	// client error
	c.JSON(http.StatusBadRequest, schema.RespErr{
		Err: err,
	})
}

func notFoundResponse(c *gin.Context, err string) {
	c.JSON(http.StatusNotFound, schema.RespErr{
		Err: err,
	})
}

func internalErrorResponse(c *gin.Context, err string) {
	// internal error
	c.JSON(http.StatusInternalServerError, schema.RespErr{
		Err: err,
	})
}

func dataResponse(c *gin.Context, dataReader *os.File, data []byte, tags []types.Tag, id string) {
	defer func() {
		if dataReader != nil {
			decFileCnt(dataReader.Name())
			dataReader.Close()
		}
	}()

	contentType := getTagValue(tags, schema.ContentType)
	if dataReader != nil {
		tmpFileName := genTmpFileName(c.ClientIP(), id)
		if dataReader.Name() != tmpFileName {
			err := os.Rename(dataReader.Name(), tmpFileName)
			if err != nil {
				internalErrorResponse(c, "data is replied")
				return
			}
			dataReader.Close()
			dataReader, err = os.Open(tmpFileName)
			if err != nil {
				internalErrorResponse(c, "data is replied")
				return
			}
		}

		incFileCnt(tmpFileName)
		err := dataRangeResponse(c, dataReader, contentType)
		if err != nil {
			internalErrorResponse(c, err.Error())
		}
	} else {
		c.Data(200, fmt.Sprintf("%s; charset=utf-8", contentType), data)
	}
}

func dataRangeResponse(c *gin.Context, dataReader *os.File, contentType string) (err error) {

	// get fileInfo
	fileInfo, err := dataReader.Stat()
	if err != nil {
		return fmt.Errorf("Error getting file info")
	}

	// fetch Range header
	rangeHeader := c.Request.Header.Get("Range")
	if rangeHeader == "" {
		c.Header("Accept-Ranges", "bytes")
		c.Header("Content-Type", contentType)
		c.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
		c.File(dataReader.Name())
		return
	}

	// parse Range header
	parts := strings.Split(rangeHeader, "=")
	if len(parts) != 2 || parts[0] != "bytes" {
		return fmt.Errorf("Invalid Range header")
	}

	ranges := strings.Split(parts[1], "-")
	if len(ranges) != 2 {
		return fmt.Errorf("Invalid Range header")
	}

	// parse Range start-end
	var start, end int64

	if ranges[0] == "" {
		end, err = strconv.ParseInt(ranges[1], 10, 64)
		if err != nil {
			end = fileInfo.Size() - 1
		}
		start = fileInfo.Size() - end
		end = fileInfo.Size() - 1
	} else if ranges[1] == "" {
		start, err = strconv.ParseInt(ranges[0], 10, 64)
		if err != nil {
			start = 0
		}
		end = fileInfo.Size() - 1
	} else {
		start, err = strconv.ParseInt(ranges[0], 10, 64)
		if err != nil {
			start = 0
		}
		end, err = strconv.ParseInt(ranges[1], 10, 64)
		if err != nil {
			end = fileInfo.Size() - 1
		}
	}

	// calculate Range size
	contentLength := end - start + 1

	// set header
	c.Header("Content-Type", contentType)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
	c.Header("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(fileInfo.Size(), 10))
	c.Status(http.StatusPartialContent)

	// send Range data
	_, err = dataReader.Seek(start, 0)
	if err != nil {
		return fmt.Errorf("error seeking file, err: %v", err)
	}

	buffer := make([]byte, 1024*1024)
	for contentLength > 0 {
		size := int64(len(buffer))
		if size > contentLength {
			size = contentLength
		}

		n, err := dataReader.Read(buffer[:size])
		if err != nil {
			return fmt.Errorf("Error reading file")
		}

		c.Writer.Write(buffer[:n])
		contentLength -= int64(n)
	}
	return nil
}

func genTmpFileName(ip, itemId string) string {
	return fmt.Sprintf("%s/%s-read-%s", schema.TmpFileDir, ip, itemId)
}

func existTmpFile(tmpFileName string) bool {
	tmpFileMapLock.Lock()
	defer tmpFileMapLock.Unlock()
	_, ok := tmpFileMap[tmpFileName]
	return ok
}

func incFileCnt(tmpFileName string) {
	tmpFileMapLock.Lock()
	defer tmpFileMapLock.Unlock()
	tmpFileMap[tmpFileName] += 1
}

func decFileCnt(tmpFileName string) {
	tmpFileMapLock.Lock()
	defer tmpFileMapLock.Unlock()
	tmpFileMap[tmpFileName] -= 1
}

func (s *Arseeding) getApiKeyInfo(c *gin.Context) {
	address := c.Param("address")
	_, addr, err := account.IDCheck(address)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}

	detail, err := s.wdb.GetApiKeyDetailByAddress(addr)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	estimateCapDe := decimal.NewFromInt(0)
	for symbol, bal := range detail.TokenBalance {
		balDe, err := decimal.NewFromString(bal.(string))
		if err != nil {
			log.Error("decimal.NewFromString(balStr)", "err", err, "bal", bal)
			continue
		}
		perFee := s.GetPerFee(symbol)
		if perFee == nil {
			continue
		}
		tokCap := decimal.NewFromInt(types.MAX_CHUNK_SIZE).Mul(balDe).DivRound(perFee.Base.Add(perFee.PerChunk), 0)
		estimateCapDe = estimateCapDe.Add(tokCap)
	}
	if estimateCapDe.LessThan(decimal.NewFromInt(types.MAX_CHUNK_SIZE)) {
		estimateCapDe = decimal.NewFromInt(0)
	}

	tokens := make(map[string]schema.TokBal)
	for symbol, bal := range detail.TokenBalance {
		perFee := s.GetPerFee(symbol)
		if perFee == nil {
			log.Error("s.GetPerFee(symbol) not found", "symbol", symbol)
			continue
		}
		tokenTags := s.everpaySdk.SymbolToTagArr(symbol)
		if len(tokenTags) == 0 {
			continue
		}
		tokens[tokenTags[0]] = schema.TokBal{Symbol: symbol, Decimals: perFee.Decimals, Balance: bal.(string)}
	}

	c.JSON(http.StatusOK, schema.RespApiKey{
		EstimateCap: estimateCapDe.String(),
		Tokens:      tokens,
	})
}

func (s *Arseeding) getApiKey(c *gin.Context) {
	timestamp := c.Param("timestamp")
	signature := c.Param("signature")
	timestampNum, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		internalErrorResponse(c, "timestamp incorrect")
		return
	}
	now := time.Now().Unix()
	if now-timestampNum > 60 { // can not lose 60s
		internalErrorResponse(c, "timestamp expired")
		return
	}

	_, addr, err := goether.Ecrecover(accounts.TextHash([]byte(timestamp)), common.FromHex(signature))
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}

	detail, err := s.wdb.GetApiKeyDetailByAddress(addr.String())
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, detail.ApiKey)
}

func (s *Arseeding) getApikeyDepositRecords(c *gin.Context) {
	address := c.Param("address")
	_, addr, err := account.IDCheck(address)
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}

	rawId, err := strconv.ParseInt(c.DefaultQuery("rawId", "0"), 10, 64)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}
	num, err := strconv.ParseInt(c.DefaultQuery("num", "20"), 10, 64)
	if err != nil {
		errorResponse(c, err.Error())
		return
	}

	receiptTxs, err := s.wdb.GetApiKeyDepositRecords(addr, rawId, int(num))
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	respRecords := make([]schema.RespReceiptEverTx, 0, len(receiptTxs))
	for _, tx := range receiptTxs {
		perFee := s.GetPerFee(tx.Symbol)
		if perFee == nil {
			continue
		}
		respRecords = append(respRecords, schema.RespReceiptEverTx{
			RawId:     tx.RawId,
			EverHash:  tx.EverHash,
			Timestamp: tx.Nonce,
			Symbol:    tx.Symbol,
			Amount:    tx.Amount,
			Decimals:  perFee.Decimals,
		})
	}

	c.JSON(http.StatusOK, respRecords)
}

func (s *Arseeding) getRealTimeOrderStatistic(c *gin.Context) {
	result := make([]schema.Result, 0)
	data, err := s.store.GetRealTimeStatistic()
	if err != nil {
		internalErrorResponse(c, err.Error())
		return
	}
	err = json.Unmarshal(data, &result)
	c.JSON(http.StatusOK, result)
}

func (s *Arseeding) getOrderStatisticByDate(c *gin.Context) {
	start := c.Query("start")
	end := c.Query("end")
	_, err := time.Parse("20060102", start)
	if err != nil {
		errorResponse(c, "Wrong time format, what is correct is yyyyMMdd")
		return
	}
	_, err = time.Parse("20060102", end)
	if err != nil {
		errorResponse(c, "Wrong time format, what is correct is yyyyMMdd")
		return
	}
	results, err := s.wdb.GetOrderStatisticByDate(schema.Range{Start: start, End: end})
	if err != nil {
		errorResponse(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, results)
}
