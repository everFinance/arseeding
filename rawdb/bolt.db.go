package rawdb

import (
	"errors"
	"github.com/everFinance/arseeding/schema"
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

type BoltDB struct {
	Db *bolt.DB
}

func NewBoltDB(boltDirPath string) (*BoltDB, error) {
	if len(boltDirPath) == 0 {
		boltDirPath = defaultDirPath
	}
	if err := os.MkdirAll(boltDirPath, os.ModePerm); err != nil {
		return nil, err
	}

	Db, err := bolt.Open(path.Join(boltDirPath, boltName), 0660, &bolt.Options{Timeout: 2 * time.Second, InitialMmapSize: 10e6})
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, errors.New("cannot obtain database lock, database may be in use by another process")
		}
		return nil, err
	}
	Db.AllocSize = boltAllocSize
	boltDB := &BoltDB{
		Db: Db,
	}
	if err := boltDB.Db.Update(func(tx *bolt.Tx) error {
		bucketNames := []string{
			schema.ChunkBucket,
			schema.TxDataEndOffSetBucket,
			schema.TxMetaBucket,
			schema.ConstantsBucket,
			schema.TaskIdPendingPoolBucket,
			schema.TaskBucket,
			schema.BundleItemBinary,
			schema.BundleItemMeta,
			schema.BundleWaitParseArIdBucket,
			schema.BundleArIdToItemIdsBucket}
		return createBuckets(tx, bucketNames)
	}); err != nil {
		return nil, err
	}
	return boltDB, nil
}

func (s *BoltDB) Put(bucket, key string, value []byte) (err error) {
	err = s.Db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		return bkt.Put([]byte(key), value)
	})
	return
}

func (s *BoltDB) Get(bucket, key string) (data []byte, err error) {
	err = s.Db.View(func(tx *bolt.Tx) error {
		data = tx.Bucket([]byte(bucket)).Get([]byte(key))
		if data == nil {
			err = schema.ErrNotExist
			return err
		}
		return nil
	})
	return
}

func (s *BoltDB) GetAllKey(bucket string) (keys []string, err error) {
	keys = make([]string, 0)
	err = s.Db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucket)).ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	return
}

func (s *BoltDB) Delete(bucket, key string) (err error) {
	err = s.Db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucket)).Delete([]byte(key))
	})
	return
}

func (s *BoltDB) Close() (err error) {
	return s.Db.Close()
}

func createBuckets(tx *bolt.Tx, buckets []string) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
			return err
		}
	}
	return nil
}
