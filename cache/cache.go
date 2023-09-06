package cache

import "time"

type Cache struct {
	Cache ICache
}

type ICache interface {
	Set(key string, entry []byte) error

	Get(key string) ([]byte, error)
}

func NewLocalCache(allKeysExpTime time.Duration) (*Cache, error) {
	cache, err := NewBigCache(allKeysExpTime)
	if err != nil {
		return nil, err
	}
	return &Cache{Cache: cache}, nil
}
