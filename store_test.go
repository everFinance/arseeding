package arseeding

import (
	"encoding/json"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"github.com/stretchr/testify/assert"
	"os"
	"sort"
	"testing"
)

func TestAllDataEndOffset(t *testing.T) {
	dbPath := "./data/tmp.db"
	preOffset := uint64(0)
	newOffset := uint64(100)
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	err = s.SaveAllDataEndOffset(newOffset)
	assert.NoError(t, err)
	offset := s.LoadAllDataEndOffset()
	assert.Equal(t, newOffset, offset)

	err = s.RollbackAllDataEndOffset(preOffset)
	assert.NoError(t, err)
	offset = s.LoadAllDataEndOffset()
	assert.Equal(t, preOffset, offset)
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestTxMeta(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	data := []byte(schema.ConstTx)
	tx := &types.Transaction{}
	err = json.Unmarshal(data, tx)
	assert.NoError(t, err)
	err = s.SaveTxMeta(*tx)
	assert.NoError(t, err)
	flag := s.IsExistTxMeta(tx.ID)
	assert.Equal(t, true, flag)
	tx2, err := s.LoadTxMeta(tx.ID)
	assert.Equal(t, tx, tx2)
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestDataEndOffset(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	offset := uint64(100)
	dataRoot := "aaa"
	dataSize := "20"
	err = s.SaveTxDataEndOffSet(dataRoot, dataSize, offset)
	flag := s.IsExistTxDataEndOffset(dataRoot, dataSize)
	assert.Equal(t, true, flag)
	newOffset, err := s.LoadTxDataEndOffSet(dataRoot, dataSize)
	assert.Equal(t, offset, newOffset)
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestChunk(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	offset := uint64(100)
	chunk := &types.GetChunk{
		DataRoot: "aaa",
		DataSize: "12",
		DataPath: "path",
		Offset:   "sada",
		Chunk:    "chunk data",
	}
	err = s.SaveChunk(offset, *chunk)
	assert.NoError(t, err)
	newChunk, err := s.LoadChunk(offset)
	assert.NoError(t, err)
	assert.Equal(t, chunk, newChunk)
	flag := s.IsExistChunk(offset)
	assert.Equal(t, true, flag)
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestPeers(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	peerMap := map[string]int64{
		"node1": 2,
		"node2": 42,
		"node3": 1,
	}
	err = s.SavePeers(peerMap)
	assert.NoError(t, err)
	peers, err := s.LoadPeers()
	assert.NoError(t, err)
	assert.Equal(t, peerMap, peers)
	flag := s.IsExistPeers()
	assert.Equal(t, true, flag)
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestTaskPendingPool(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	taskIds := []string{"123", "456", "789"}
	for _, id := range taskIds {
		err = s.PutTaskPendingPool(id)
		assert.NoError(t, err)
	}
	ids, err := s.LoadAllPendingTaskIds()
	assert.NoError(t, err)
	sort.Strings(taskIds)
	sort.Strings(ids)
	assert.Equal(t, taskIds, ids)
	for _, id := range taskIds {
		err = s.DelPendingPoolTaskId(id)
		assert.NoError(t, err)
	}
	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestTask(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	tkId := "123"
	tk := &schema.Task{
		ArId: "12123",
	}
	err = s.SaveTask(tkId, *tk)
	assert.NoError(t, err)
	tk2, err := s.LoadTask(tkId)
	assert.NoError(t, err)
	assert.Equal(t, tk, tk2)

	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestItem(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	itemB := []byte("some data")
	item := types.BundleItem{
		SignatureType: 0,
		Id:            "123",
		ItemBinary:    nil,
	}
	itemId := item.Id
	err = s.SaveItemBinary(item)
	assert.NoError(t, err)
	flag := s.IsExistItemBinary(itemId)
	assert.Equal(t, true, flag)
	err = s.SaveItemMeta(item)
	assert.NoError(t, err)

	item2, err := s.LoadItemMeta(itemId)
	assert.NoError(t, err)
	assert.Equal(t, item, item2)
	_, data, err := s.LoadItemBinary(itemId)
	assert.NoError(t, err)
	assert.Equal(t, itemB, data)
	err = s.DelItemMeta(itemId)
	assert.NoError(t, err)
	err = s.DelItemBinary(itemId)
	assert.NoError(t, err)

	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestBundle(t *testing.T) {
	dbPath := "./data/tmp.db"
	s, err := NewBoltStore(dbPath)
	assert.NoError(t, err)
	arIds := []string{"123", "456", "789"}
	for _, id := range arIds {
		err = s.SaveWaitParseBundleArId(id)
		assert.NoError(t, err)
	}
	ids, err := s.LoadWaitParseBundleArIds()
	sort.Strings(arIds)
	sort.Strings(ids)
	assert.Equal(t, arIds, ids)

	for _, id := range arIds {
		err = s.DelParsedBundleArId(id)
		assert.NoError(t, err)
	}
	itemIds := arIds
	arId := "aaaaaa"
	err = s.SaveArIdToItemIds(arId, itemIds)
	assert.NoError(t, err)
	flag := s.ExistArIdToItemIds(arId)
	assert.Equal(t, true, flag)
	ids, err = s.LoadArIdToItemIds(arId)
	assert.NoError(t, err)
	sort.Strings(ids)
	sort.Strings(itemIds)
	assert.Equal(t, itemIds, ids)

	err = os.RemoveAll(dbPath)
	assert.NoError(t, err)
}

func TestIntByte(t *testing.T) {
	v := uint64(233333)
	str := itob(v)
	v2 := btoi(str)
	assert.Equal(t, v, v2)
}
