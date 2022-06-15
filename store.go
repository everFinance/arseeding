package arseeding

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	bolt "go.etcd.io/bbolt"
	"os"
	"path"
	"time"
)

const (
	boltAllocSize = 8 * 1024 * 1024

	defaultDirPath = "./data/bolt"
	boltName       = "seed.db"
)

var (
	// bucket
	ChunkBucket           = []byte("chunk-bucket")              // key: chunkStartOffset, val: chunk
	TxDataEndOffSetBucket = []byte("tx-data-end-offset-bucket") // key: dataRoot+dataSize; val: txDataEndOffSet
	TxMetaBucket          = []byte("tx-meta-bucket")            // key: txId, val: arTx; not include data
	ConstantsBucket       = []byte("constants-bucket")

	// tasks
	TaskIdPendingPoolBucket = []byte("task-pending-pool-bucket") // key: taskId(taskType+"-"+arId), val: "0x01"
	TaskBucket              = []byte("task-bucket")              // key: taskId(taskType+"-"+arId), val: task

	// bundle bucketName
	BundleItemBinary = []byte("bundle-item-binary")
	BundleItemMeta   = []byte("bundle-item-meta")

	// parse arTx data to bundle items
	BundleWaitParseArIdBucket = []byte("bundle-wait-parse-arId-bucket") // key: arId, val: "0x01"
	BundleArIdToItemIdsBucket = []byte("bundle-arId-to-itemIds-bucket") // key: arId, val: json.marshal(itemIds)

)

type Store struct {
	BoltDb *bolt.DB
}

func NewStore(boltDirPath string) (*Store, error) {
	if len(boltDirPath) == 0 {
		boltDirPath = defaultDirPath
	}
	if err := os.MkdirAll(boltDirPath, os.ModePerm); err != nil {
		return nil, err
	}

	boltDB, err := bolt.Open(path.Join(boltDirPath, boltName), 0660, &bolt.Options{Timeout: 2 * time.Second, InitialMmapSize: 10e6})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}
	boltDB.AllocSize = boltAllocSize

	kv := &Store{
		BoltDb: boltDB,
	}

	// create bucket
	if err := kv.BoltDb.Update(func(tx *bolt.Tx) error {
		bucketNames := [][]byte{
			ChunkBucket,
			TxDataEndOffSetBucket,
			TxMetaBucket,
			ConstantsBucket,
			TaskIdPendingPoolBucket,
			TaskBucket,
			BundleItemBinary,
			BundleItemMeta,
			BundleWaitParseArIdBucket,
			BundleArIdToItemIdsBucket}
		return createBuckets(tx, bucketNames...)
	}); err != nil {
		return nil, err
	}

	return kv, nil
}

func (s *Store) Close() error {
	return s.BoltDb.Close()
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return err
		}
	}
	return nil
}

// about tx

func (s *Store) SaveAllDataEndOffset(allDataEndOffset uint64, dbTx *bolt.Tx) (err error) {
	if dbTx == nil {
		dbTx, err = s.BoltDb.Begin(true)
		if err != nil {
			return
		}
		defer dbTx.Commit()
	}
	key := []byte("allDataEndOffset")
	val := itob(allDataEndOffset)

	bkt, err := dbTx.CreateBucketIfNotExists(ConstantsBucket)
	if err != nil {
		return err
	}
	return bkt.Put(key, val)
}

func (s *Store) LoadAllDataEndOffset() (offset uint64) {
	key := []byte("allDataEndOffset")
	_ = s.BoltDb.View(func(tx *bolt.Tx) error {
		val := tx.Bucket(ConstantsBucket).Get(key)
		if val == nil {
			offset = 0
		} else {
			offset = btoi(val)
		}
		return nil
	})
	return
}

func (s *Store) SaveTxMeta(arTx types.Transaction) error {
	arTx.Data = "" // only store tx meta, not include data
	key := []byte(arTx.ID)
	val, err := json.Marshal(&arTx)
	if err != nil {
		return err
	}
	return s.BoltDb.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(TxMetaBucket)
		return bkt.Put(key, val)
	})
}

func (s *Store) LoadTxMeta(arId string) (arTx *types.Transaction, err error) {
	key := []byte(arId)
	arTx = &types.Transaction{}
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		val := tx.Bucket(TxMetaBucket).Get(key)
		if val == nil {
			return ErrNotExist
		} else {
			err = json.Unmarshal(val, arTx)
			return err
		}
	})
	return
}

func (s *Store) IsExistTxMeta(arId string) bool {
	_, err := s.LoadTxMeta(arId)
	if err == ErrNotExist {
		return false
	}
	return true
}

func (s *Store) SaveTxDataEndOffSet(dataRoot, dataSize string, txDataEndOffset uint64, dbTx *bolt.Tx) (err error) {
	if dbTx == nil {
		dbTx, err = s.BoltDb.Begin(true)
		if err != nil {
			return
		}
		defer dbTx.Commit()
	}

	bkt, err := dbTx.CreateBucketIfNotExists(TxDataEndOffSetBucket)
	if err != nil {
		return err
	}
	return bkt.Put(generateOffSetKey(dataRoot, dataSize), itob(txDataEndOffset))
}

func (s *Store) LoadTxDataEndOffSet(dataRoot, dataSize string) (txDataEndOffset uint64, err error) {
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(TxDataEndOffSetBucket)
		val := bkt.Get(generateOffSetKey(dataRoot, dataSize))
		if val == nil {
			return ErrNotExist
		} else {
			txDataEndOffset = btoi(val)
		}
		return nil
	})
	return
}

func (s *Store) IsExistTxDataEndOffset(dataRoot, dataSize string) bool {
	_, err := s.LoadTxDataEndOffSet(dataRoot, dataSize)
	if err == ErrNotExist {
		return false
	}
	return true
}

func (s *Store) SaveChunk(chunkStartOffset uint64, chunk types.GetChunk) error {
	chunkJs, err := chunk.Marshal()
	if err != nil {
		return err
	}
	err = s.BoltDb.Update(func(tx *bolt.Tx) error {
		chunkBkt := tx.Bucket(ChunkBucket)
		if err := chunkBkt.Put(itob(chunkStartOffset), chunkJs); err != nil {
			return err
		}
		return nil
	})

	return err
}

func (s *Store) LoadChunk(chunkStartOffset uint64) (chunk *types.GetChunk, err error) {
	chunk = &types.GetChunk{}
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		chunkBkt := tx.Bucket(ChunkBucket)
		val := chunkBkt.Get(itob(chunkStartOffset))
		if val == nil {
			err = ErrNotExist
			return err
		} else {
			err = json.Unmarshal(val, chunk)
			return err
		}
	})

	return chunk, err
}

func (s *Store) IsExistChunk(chunkStartOffset uint64) bool {
	_, err := s.LoadChunk(chunkStartOffset)
	if err == ErrNotExist {
		return false
	}
	return true
}

func (s *Store) SavePeers(peers []string) error {
	peersB, err := json.Marshal(peers)
	key := []byte("peer-list")
	if err != nil {
		return err
	}
	err = s.BoltDb.Update(func(tx *bolt.Tx) error {
		chunkBkt := tx.Bucket(ConstantsBucket)
		if err := chunkBkt.Put(key, peersB); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (s *Store) LoadPeers() (peers []string, err error) {
	key := []byte("peer-list")
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(ConstantsBucket)
		val := bkt.Get(key)
		if val == nil {
			return ErrNotExist
		}
		err = json.Unmarshal(val, &peers)
		return err
	})
	return
}

func (s *Store) IsExistPeers() bool {
	_, err := s.LoadPeers()
	if err == ErrNotExist {
		return false
	}
	return true
}

// itob returns an 64-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 64)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func btoi(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func generateOffSetKey(dataRoot, dataSize string) []byte {
	hash := sha256.Sum256([]byte(dataRoot + dataSize))
	return hash[:]
}

// about tasks

func (s *Store) PutTaskPendingPool(taskId string) error {
	return s.BoltDb.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(TaskIdPendingPoolBucket).Put([]byte(taskId), []byte("0x01"))
	})
}

func (s *Store) LoadAllPendingTaskIds() ([]string, error) {
	taskIds := make([]string, 0)
	err := s.BoltDb.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(TaskIdPendingPoolBucket)

		return bkt.ForEach(func(k, v []byte) error {
			taskIds = append(taskIds, string(k))
			return nil
		})
	})
	return taskIds, err
}

func (s *Store) DelPendingPoolTaskId(taskId string) error {
	return s.BoltDb.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(TaskIdPendingPoolBucket).Delete([]byte(taskId))
	})
}

func (s *Store) SaveTask(taskId string, tk schema.Task) error {
	val, err := json.Marshal(&tk)
	if err != nil {
		return err
	}

	return s.BoltDb.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(TaskBucket).Put([]byte(taskId), val)
	})
}

func (s *Store) LoadTask(taskId string) (tk *schema.Task, err error) {
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		val := tx.Bucket(TaskBucket).Get([]byte(taskId))
		if val == nil {
			return ErrNotExist
		} else {
			err = json.Unmarshal(val, tk)
			return err
		}
	})
	return
}

// about bundle

func (s *Store) SaveItemBinary(itemId string, itemBinary []byte, dbTx *bolt.Tx) (err error) {
	if dbTx == nil {
		dbTx, err = s.BoltDb.Begin(true)
		if err != nil {
			return
		}
		defer dbTx.Commit()
	}

	bkt, err := dbTx.CreateBucketIfNotExists(BundleItemBinary)
	if err != nil {
		return err
	}
	return bkt.Put([]byte(itemId), itemBinary)
}

func (s *Store) IsExistItemBinary(itemId string) bool {
	_, err := s.LoadItemBinary(itemId)
	if err == ErrNotExist {
		return false
	}
	return true
}

func (s *Store) LoadItemBinary(itemId string) (itemBinary []byte, err error) {
	key := []byte(itemId)

	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		itemBinary = tx.Bucket(BundleItemBinary).Get(key)
		if itemBinary == nil {
			return ErrNotExist
		}
		return nil
	})
	return
}

func (s *Store) DelItemBinary(itemId string, dbTx *bolt.Tx) (err error) {
	if dbTx == nil {
		dbTx, err = s.BoltDb.Begin(true)
		if err != nil {
			return
		}
		defer dbTx.Commit()
	}
	key := []byte(itemId)
	return dbTx.Bucket(BundleItemBinary).Delete(key)
}

func (s *Store) SaveItemMeta(item types.BundleItem, dbTx *bolt.Tx) (err error) {
	if dbTx == nil {
		dbTx, err = s.BoltDb.Begin(true)
		if err != nil {
			return
		}
		defer dbTx.Commit()
	}

	bkt, err := dbTx.CreateBucketIfNotExists(BundleItemMeta)
	if err != nil {
		return err
	}
	item.Data = "" // without data
	meta, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return bkt.Put([]byte(item.Id), meta)
}

func (s *Store) LoadItemMeta(itemId string) (meta types.BundleItem, err error) {
	key := []byte(itemId)
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		metaBy := tx.Bucket(BundleItemMeta).Get(key)
		if metaBy == nil {
			return ErrNotExist
		}
		return json.Unmarshal(metaBy, &meta)
	})
	return
}

func (s *Store) DelItemMeta(itemId string, dbTx *bolt.Tx) (err error) {
	if dbTx == nil {
		dbTx, err = s.BoltDb.Begin(true)
		if err != nil {
			return
		}
		defer dbTx.Commit()
	}
	key := []byte(itemId)
	return dbTx.Bucket(BundleItemMeta).Delete(key)
}

// bundle items to arTx

func (s *Store) SaveWaitParseBundleArId(arId string) error {
	return s.BoltDb.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(BundleWaitParseArIdBucket).Put([]byte(arId), []byte("0x01"))
	})
}

func (s *Store) LoadWaitParseBundleArIds() (arIds []string, err error) {
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		return tx.Bucket(BundleWaitParseArIdBucket).ForEach(func(k, v []byte) error {
			arIds = append(arIds, string(k))
			return nil
		})
	})
	return
}

func (s *Store) DelParsedBundleArId(arId string) error {
	return s.BoltDb.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(BundleWaitParseArIdBucket).Delete([]byte(arId))
	})
}

func (s *Store) SaveArIdToItemIds(arId string, itemIds []string) error {
	return s.BoltDb.Update(func(tx *bolt.Tx) error {
		itemIdsJs, err := json.Marshal(itemIds)
		if err != nil {
			return err
		}
		return tx.Bucket(BundleArIdToItemIdsBucket).Put([]byte(arId), itemIdsJs)
	})
}

func (s *Store) LoadArIdToItemIds(arId string) (itemIds []string, err error) {
	err = s.BoltDb.View(func(tx *bolt.Tx) error {
		val := tx.Bucket(BundleArIdToItemIdsBucket).Get([]byte(arId))
		if val == nil {
			return ErrNotExist
		} else {
			return json.Unmarshal(val, &itemIds)
		}
	})
	return
}

func (s *Store) ExistArIdToItemIds(arId string) bool {
	_, err := s.LoadArIdToItemIds(arId)
	if err == ErrNotExist {
		return false
	}
	return true
}
