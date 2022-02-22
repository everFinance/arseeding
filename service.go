package arseeding

import (
	"errors"
	"fmt"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"strconv"
)

func (s *Server) processSubmitChunk(chunk types.GetChunk) error {
	// 1. verify chunk
	err, ok := verifyChunk(chunk)
	if err != nil || !ok {
		log.Error("verifyChunk(chunk) failed", "err", err, "chunk", chunk)
		return fmt.Errorf("verifyChunk error:%v", err)
	}

	s.submitLocker.Lock()
	defer s.submitLocker.Unlock()

	// 2. check TxDataEndOffset exist
	if !s.store.IsExistTxDataEndOffset(chunk.DataRoot, chunk.DataSize) {
		// add TxDataEndOffset
		if err := s.syncAddTxDataEndOffset(chunk.DataRoot, chunk.DataSize); err != nil {
			log.Error("syncAddTxDataEndOffset(s.store,chunk.DataRoot,chunk.DataSize)", "err", err, "chunk", chunk)
			return err
		}
	}

	// 3. store chunk
	if err := storeChunk(chunk, s.store); err != nil {
		log.Error("storeChunk(chunk,s.store)", "err", err, "chunk", chunk)
		return err
	}

	return nil
}

func setTxDataChunks(arTx types.Transaction, txData []byte, db *Store) error {
	if len(txData) == 0 {
		return errors.New("tx data not be null")
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
			log.Error("storeChunk(*chunk,s.store)", "err", err, "chunk", *chunk)
			return err
		}
	}
	return nil
}

func generateChunks(arTxMeta types.Transaction, data []byte) ([]*types.GetChunk, error) {
	if len(data) == 0 {
		return nil, errors.New("data can not null")
	}
	utils.PrepareChunks(&arTxMeta, data)

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

func (s *Server) syncAddTxDataEndOffset(dataRoot, dataSize string) error {
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

	// must use tx db
	boltTx, err := s.store.BoltDb.Begin(true)
	if err != nil {
		log.Error("s.store.BoltDb.Begin(true)", "err", err)
		return err
	}
	if err := s.store.SaveAllDataEndOffset(newEndOffset, boltTx); err != nil {
		boltTx.Rollback()
		log.Error("s.store.SaveAllDataEndOffset(newEndOffset)", "err", err)
		return err
	}
	// SaveTxDataEndOffSet
	if err := s.store.SaveTxDataEndOffSet(dataRoot, dataSize, newEndOffset, boltTx); err != nil {
		boltTx.Rollback()
		return err
	}
	// commit
	if err := boltTx.Commit(); err != nil {
		boltTx.Rollback()
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
