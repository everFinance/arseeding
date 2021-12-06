package seeding

import (
	"errors"
	bolt "go.etcd.io/bbolt"
	"os"
	"path"
	"time"
)

const (
	boltAllocSize = 8 * 1024 * 1024

	dirPath  = "./store"
	boltName = "seed.db"
)

type Store struct {
	BoltDb *bolt.DB
}

func NewStore(bucketNames ...[]byte) (*Store, error) {
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return nil, err
	}

	boltDB, err := bolt.Open(path.Join(dirPath, boltName), 0660, &bolt.Options{Timeout: 1 * time.Second, InitialMmapSize: 10e6})
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
		return createBuckets(tx, bucketNames...)
	}); err != nil {
		return nil, err
	}

	return kv, nil
}

func createBuckets(tx *bolt.Tx, buckets ...[]byte) error {
	for _, bucket := range buckets {
		if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) IsExist(arid string) bool {
	// TODO
	return false
}

func (s *Store) SaveTx(artx, offset interface{}) error {
	// TODO
	return nil
}

func (s *Store) LoadTx(arid string) (interface{}, error) {
	// TODO
	return nil, nil
}

func (s *Store) SaveChunks(chunk interface{}) error {

	// TODO
	return nil
}

func (s *Store) LoadChunk(offset interface{}) (interface{}, error) {
	// TODO
	return nil, nil
}
