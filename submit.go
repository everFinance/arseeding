package arseeding

import (
	"errors"
	"fmt"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"gorm.io/gorm"
	"math/big"
	"strconv"
)

func (s *Arseeding) SaveSubmitChunk(chunk types.GetChunk) error {
	// 1. verify chunk
	err, ok := verifyChunk(chunk)
	if err != nil || !ok {
		log.Error("verifyChunk(chunk) failed", "err", err, "chunk.DataRoot", chunk.DataRoot)
		return fmt.Errorf("verifyChunk error:%v", err)
	}

	s.submitLocker.Lock()
	defer s.submitLocker.Unlock()

	// 2. check TxDataEndOffset exist
	if !s.store.IsExistTxDataEndOffset(chunk.DataRoot, chunk.DataSize) {
		// add TxDataEndOffset
		if err := s.syncAddTxDataEndOffset(chunk.DataRoot, chunk.DataSize); err != nil {
			log.Error("syncAddTxDataEndOffset(s.store,chunk.DataRoot,chunk.DataSize)", "err", err, "chunk.DataRoot", chunk.DataRoot)
			return err
		}
	}

	// 3. store chunk
	if err := storeChunk(chunk, s.store); err != nil {
		log.Error("storeChunk(chunk,s.store)", "err", err, "chunk.DataRoot", chunk.DataRoot)
		return err
	}

	return nil
}

func (s *Arseeding) SaveSubmitTx(arTx types.Transaction) error {
	// 1. verify ar tx
	if err := utils.VerifyTransaction(arTx); err != nil {
		log.Error("utils.VerifyTransaction(arTx)", "err", err, "arTx", arTx.ID)
		return err
	}

	// 2. check meta exist
	if s.store.IsExistTxMeta(arTx.ID) {
		return schema.ErrExistTx
	}

	// 3. save tx meta
	if err := s.store.SaveTxMeta(arTx); err != nil {
		log.Error("s.store.SaveTxMeta(arTx)", "err", err, "arTx", arTx.ID)
		return err
	}

	s.submitLocker.Lock()
	defer s.submitLocker.Unlock()

	// 4. check whether update allDataEndOffset
	if s.store.IsExistTxDataEndOffset(arTx.DataRoot, arTx.DataSize) {
		return nil
	}
	// add txDataEndOffset
	if err := s.syncAddTxDataEndOffset(arTx.DataRoot, arTx.DataSize); err != nil {
		log.Error("syncAddTxDataEndOffset(s.store,arTx.DataRoot,arTx.DataSize)", "err", err, "arTx", arTx.ID)
		return err
	}

	// 5. save tx data chunk if exist
	if len(arTx.Data) > 0 {
		// set chunks
		dataBy, err := utils.Base64Decode(arTx.Data)
		if err != nil {
			log.Error("utils.Base64Decode(arTx.Data)", "err", err, "arTx", arTx.ID)
			return err
		}
		if err := setTxDataChunks(arTx, dataBy, s.store); err != nil {
			return err
		}
	}
	// parse ANS-104 bundle data
	if isBundleTx(arTx.Tags) {
		return s.store.SaveWaitParseBundleArId(arTx.ID)
	}
	return nil
}

func (s *Arseeding) FetchAndStoreTx(arId string) (err error) {
	// 1. sync arTxMeta
	arTxMeta := &types.Transaction{}
	arTxMeta, err = s.store.LoadTxMeta(arId)
	if err != nil {
		// get txMeta from arweave network
		arTxMeta, err = s.arCli.GetUnconfirmedTx(arId) // this api can return all tx (unconfirmed and confirmed)
		if err != nil {
			// get tx from peers
			arTxMeta, err = s.taskMg.GetUnconfirmedTxFromPeers(arId, schema.TaskTypeSync, s.cache.GetPeers())
			if err != nil {
				return err
			}
		}
	}
	if len(arTxMeta.ID) == 0 { // arTxMeta can not be null
		return fmt.Errorf("get arTxMeta failed; arId: %s", arId)
	}

	// if arTxMeta not exist local, store it
	if !s.store.IsExistTxMeta(arId) {
		// store txMeta
		if err := s.store.SaveTxMeta(*arTxMeta); err != nil {
			log.Error("s.store.SaveTxMeta(arTx)", "err", err, "arTx", arTxMeta.ID)
			return err
		}

	}

	if !s.store.IsExistTxDataEndOffset(arTxMeta.DataRoot, arTxMeta.DataSize) {
		// store txDataEndOffset
		if err := s.syncAddTxDataEndOffset(arTxMeta.DataRoot, arTxMeta.DataSize); err != nil {
			log.Error("s.syncAddTxDataEndOffset(arTxMeta.DataRoot,arTxMeta.DataSize)", "err", err, "arId", arId)
			return err
		}
	}

	// 2. sync data
	size, ok := new(big.Int).SetString(arTxMeta.DataSize, 10)
	if !ok {
		return fmt.Errorf("new(big.Int).SetString(arTxMeta.DataSize,10) failed; dataSize: %s", arTxMeta.DataSize)
	}

	if size.Cmp(big.NewInt(0)) <= 0 {
		// not need to sync tx data
		return nil
	}

	// get data
	var data []byte
	data, err = getArTxData(arTxMeta.DataRoot, arTxMeta.DataSize, s.store) // get data from local
	if err == nil {
		return nil // local exist data
	}
	// need get tx data from arweave network
	data, err = s.arCli.GetTransactionDataByGateway(arId)
	if err != nil {
		data, err = s.taskMg.GetTxDataFromPeers(arId, schema.TaskTypeSync, s.cache.GetPeers())
		if err != nil {
			log.Error("get data failed", "err", err, "arId", arId)
			return err
		}
	}

	// store data to local
	if err = setTxDataChunks(*arTxMeta, data, s.store); err != nil {
		return err
	}

	// process manifest
	tags, _ := utils.TagsDecode(arTxMeta.Tags)
	if s.EnableManifest && getTagValue(tags, schema.ContentType) == schema.ManifestType {
		mfUrl := expectedTxSandbox(arId)
		// insert new record
		if _, err = s.wdb.GetManifestId(mfUrl); err == gorm.ErrRecordNotFound {
			if err = s.wdb.InsertManifest(schema.Manifest{
				ManifestUrl: mfUrl,
				ManifestId:  arId,
			}); err != nil {
				log.Error("s.wdb.InsertManifest(res)", "err", err, "arId", arId)
				return err
			}
		}
	}

	// parse ANS-104 bundle data
	if isBundleTx(arTxMeta.Tags) {
		return s.store.SaveWaitParseBundleArId(arId)
	}
	return nil
}

func (s *Arseeding) syncAddTxDataEndOffset(dataRoot, dataSize string) error {
	s.endOffsetLocker.Lock()
	defer s.endOffsetLocker.Unlock()

	if s.store.IsExistTxDataEndOffset(dataRoot, dataSize) {
		return nil
	}

	// update allDataEndOffset
	txSize, err := strconv.ParseUint(dataSize, 10, 64)
	if err != nil {
		log.Error("strconv.ParseUint(arTx.DataSize,10,64)", "err", err)
		return err
	}
	curEndOffset := s.store.LoadAllDataEndOffset()
	newEndOffset := curEndOffset + txSize

	return s.store.AtomicSyncDataEndOffset(curEndOffset, newEndOffset, dataRoot, dataSize)
}

func setTxDataChunks(arTx types.Transaction, txData []byte, db *Store) error {
	if len(txData) == 0 {
		return schema.ErrNullData
	}
	chunks, err := generateChunks(arTx, txData)
	if err != nil {
		log.Error("generateChunks(arTx, dataBy)", "err", err, "data", arTx.Data, "arTx", arTx.ID)
		return err
	}
	// store chunk
	for _, chunk := range chunks {
		if chunk.DataRoot != arTx.DataRoot {
			log.Error("chunk dataRoot not equal tx dataRoot", "chunkRoot", chunk.DataRoot, "txRoot", arTx.DataRoot)
			return errors.New("chunk dataRoot not equal tx dataRoot")
		}

		if err := storeChunk(*chunk, db); err != nil {
			log.Error("storeChunk(*chunk,s.store)", "err", err)
			return err
		}
	}
	return nil
}

func generateChunks(arTxMeta types.Transaction, data []byte) ([]*types.GetChunk, error) {
	if len(data) == 0 {
		return nil, schema.ErrNullData
	}
	utils.PrepareChunks(&arTxMeta, data, len(data))

	chunks := make([]*types.GetChunk, 0, len(arTxMeta.Chunks.Chunks))
	for i := 0; i < len(arTxMeta.Chunks.Chunks); i++ {
		chunk, err := utils.GetChunk(arTxMeta, i, data)
		if err != nil {
			log.Error("utils.GetChunk(arTxMeta,i,data)", "err", err, "i", i, "arId", arTxMeta.ID)
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func storeChunk(chunk types.GetChunk, db *Store) error {
	// generate chunkStartOffset
	txDataEndOffset, err := db.LoadTxDataEndOffSet(chunk.DataRoot, chunk.DataSize)
	if err != nil {
		log.Error("db.LoadTxDataEndOffSet(chunk.DataRoot,chunk.DataSize)", "err", err, "root", chunk.DataRoot, "size", chunk.DataSize)
		return err
	}
	txSize, err := strconv.ParseUint(chunk.DataSize, 10, 64)
	if err != nil {
		return err
	}
	txDataStartOffset := txDataEndOffset - txSize + 1

	offset, err := strconv.ParseUint(chunk.Offset, 10, 64)
	if err != nil {
		return err
	}
	chunkEndOffset := txDataStartOffset + offset
	chunkDataBy, err := utils.Base64Decode(chunk.Chunk)
	if err != nil {
		return err
	}
	chunkStartOffset := chunkEndOffset - uint64(len(chunkDataBy)) + 1
	// save
	if err := db.SaveChunk(chunkStartOffset, chunk); err != nil {
		log.Error("s.store.SaveChunk(chunkStartOffset, *chunk)", "err", err)
		return err
	}
	return nil
}

func verifyChunk(chunk types.GetChunk) (err error, ok bool) {
	dataRoot, err := utils.Base64Decode(chunk.DataRoot)
	if err != nil {
		return
	}
	offset, err := strconv.Atoi(chunk.Offset)
	if err != nil {
		return
	}
	dataSize, err := strconv.Atoi(chunk.DataSize)
	if err != nil {
		return
	}
	path, err := utils.Base64Decode(chunk.DataPath)
	if err != nil {
		return
	}
	_, ok = utils.ValidatePath(dataRoot, offset, 0, dataSize, path)
	return
}

func isBundleTx(txTags []types.Tag) bool {
	var (
		fcount = 0
		vcount = 0
	)
	tags, _ := utils.TagsDecode(txTags)
	for _, tg := range tags {
		if tg.Name == "Bundle-Format" && tg.Value == "binary" {
			fcount++
		}
		if tg.Name == "Bundle-Version" && tg.Value == "2.0.0" {
			vcount++
		}
	}
	return fcount != 0 && vcount != 0
}
